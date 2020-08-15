/*
* Specialized version of exec that is intended to run uploaded code.
* It does the following:
*   - Checks to make sure that the exam has already been uploaded AND compiled.
*   - Runs the make check target in the corresponding Makefile.
*/

package handlers

import (
  "strings"
	"io"
	"log"
	"net/http"
  "encoding/json"
  "fmt"

	"github.com/gorilla/mux"
	"github.com/play-with-docker/play-with-docker/storage"
)

func ExamRun(rw http.ResponseWriter, req *http.Request) {
  vars := mux.Vars(req)
  sessionId := vars["sessionId"]
  instanceName := vars["instanceName"]

  s, err := core.SessionGet(sessionId)
  if err == storage.NotFoundError {
    rw.WriteHeader(http.StatusNotFound)
    return
  } else if err != nil {
    rw.WriteHeader(http.StatusInternalServerError)
    return
  }
  i := core.InstanceGet(s, instanceName)

  examName := req.URL.Query().Get("examname")

  // Step 1: Make sure the uploaded code has been compiled.
  var lsCmd = fmt.Sprintf(
    `{ "command": ["ls", "exams/%s/%s"] }`,
     examName, examName + "_submission")

  var er1 execRequest // from exec.go handler

  err = json.NewDecoder(strings.NewReader(lsCmd)).Decode(&er1)
  if err != nil {
    log.Fatal(err)
    rw.WriteHeader(http.StatusBadRequest)
    return
  }

  cmdout, err := core.InstanceExecOutput(i, er1.Cmd)

  if err != nil {
    log.Printf("Error executing command; error: %s, cmdout: %s", err, cmdout)
    rw.WriteHeader(http.StatusInternalServerError)
    return
  }

  buf := new(strings.Builder)
  io.Copy(buf, cmdout)

  // If ls returns "No such file or directory" for the test name executable,
  // then it has not been successfully compiled. Respond with 404 NOT FOUND.
  // Otherwise, we can continue to step 2.
  if strings.Contains(buf.String(), "No such file or directory") {
    rw.WriteHeader(http.StatusNotFound)
    return
  }

  // Step 2: Run the make check target
  // The -s flag could be used instead of --no-print-directory, but -s strips a lot more.
  var makeCheckCmd = fmt.Sprintf(`{ "command": ["make", "check", "--no-print-directory", "-C", "%s"] }`, "exams/" + examName)

  var er2 execRequest // from exec.go handler

  err = json.NewDecoder(strings.NewReader(makeCheckCmd)).Decode(&er2)
  if err != nil {
    log.Fatal(err)
    rw.WriteHeader(http.StatusInternalServerError)
    return
  }

  // Step 2.5: Check if make compilation was successful

  cmdout2, err := core.InstanceExecOutput(i, er2.Cmd)

  if err != nil {
    log.Printf("Error executing command; error: %s, cmdout: %s", err, cmdout)
    rw.WriteHeader(http.StatusInternalServerError)
    return
  }

  rw.Header().Set("content-type", "text/html")

  if _,err = io.Copy(rw, cmdout2); err != nil {
    log.Println(err)
    rw.WriteHeader(http.StatusInternalServerError)
    return
  }

  rw.WriteHeader(http.StatusOK)
  return

}
