/*
* Specialized version of file_upload that is intended to upload code.
* It does the following:
*   - Uploads the code to the instance via the multipart form data
*   - Compiles the code
* HTTP Response back contains the compilation status (success or fail), and
* any errors if fail.
* NOTE: We do not suppress compiler warnings, this is the job of the Makefile.
*/

package handlers

import (
  "strings"
	"io"
	"log"
	"net/http"
  "encoding/json"
  "fmt"

  "github.com/play-with-docker/play-with-docker/config"
	"github.com/gorilla/mux"
	"github.com/play-with-docker/play-with-docker/storage"
)

func ExamUploadCompile(rw http.ResponseWriter, req *http.Request) {
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

  compiler := req.URL.Query().Get("compiler")
  var examEndpoint string

  if compiler == "rose" {
    log.Println("Using ROSE endpoint.")
    examEndpoint = config.RoseExamEndpoint
  } else if compiler == "llvm" {
    log.Println("Using LLVM endpoint.")
    examEndpoint = config.LLVMExamEndpoint
  } else {
    rw.WriteHeader(http.StatusBadRequest)
    return
  }

  // Step 0: Grab the necessary resources from the hosted endpoint
  var cloneCmd = fmt.Sprintf(
    `{ "command":
    ["git", "clone", "%s", "."]
    }`,
    examEndpoint)

  var er1 execRequest // from exec.go handler

  err = json.NewDecoder(strings.NewReader(cloneCmd)).Decode(&er1)
  if err != nil {
    log.Fatal(err)
    rw.WriteHeader(http.StatusBadRequest)
    return
  }

  code, err := core.InstanceExec(i, er1.Cmd)

  if err != nil {
    log.Printf("Error executing command; code: %s, error: %s", code, err)
    rw.WriteHeader(http.StatusInternalServerError)
    return
  }

  log.Printf("Obtained exam resources from endpoint.")

  // Step 1: upload the code to the instance

	// allow up to 32 MB which is the default

  // Grab the exam name that we will use to match the directory at the host endpoint
  examName := req.URL.Query().Get("examname")

	red, err := req.MultipartReader()
	if err != nil {
		log.Println(err)
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	path := req.URL.Query().Get("path") + "/exams/" + examName

	for {
		p, err := red.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Println(err)
			continue
		}

		if p.FileName() == "" {
			continue
		}
		err = core.InstanceUploadFromReader(i, p.FileName(), path, p)
		if err != nil {
			log.Println(err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		log.Printf("Uploaded [%s] to [%s]\n", p.FileName(), i.Name)
  }

  // Step 2: Compile via make

  var makeCmd = fmt.Sprintf(`{ "command": ["make", "-B", "-C", "%s"] }`, path)

  var er2 execRequest // from exec.go handler

  err = json.NewDecoder(strings.NewReader(makeCmd)).Decode(&er2)
  if err != nil {
    log.Fatal(err)
    rw.WriteHeader(http.StatusInternalServerError)
    return
  }

  // Step 2.5: Check if make compilation was successful

  cmdout, err := core.InstanceExecOutput(i, er2.Cmd)

  if err != nil {
    log.Printf("Error executing command; error: %s, cmdout: %s", err, cmdout)
    rw.WriteHeader(http.StatusInternalServerError)
    return
  }

  rw.Header().Set("content-type", "text/html")

  buf := new(strings.Builder)
  io.Copy(buf, cmdout)
  log.Println(buf.String())

  // Parses the make response the word "Error" or "Stop" that shows when make fails.
  // If there is an error, send it back with the HTTP response. Else, send success.
  // NOTE: We send back a 502 (Bad Gateway) on compile error, along with the error(s).
  // Client should handle this appropriately (check if 502 and print error)
  if strings.Contains(buf.String(), "Error") || strings.Contains(buf.String(), "Stop") {
    comperr := strings.NewReader(buf.String())
    rw.WriteHeader(http.StatusBadGateway)
    if _,err = io.Copy(rw, comperr); err != nil {
      log.Println(err)
      rw.WriteHeader(http.StatusInternalServerError)
      return
    }
    return
  } else {
    suc := strings.NewReader("BUILD SUCCESSFUL.")
    if _,err = io.Copy(rw, suc); err != nil {
      log.Println(err)
      rw.WriteHeader(http.StatusInternalServerError)
      return
    }
  }

  rw.WriteHeader(http.StatusOK)
  return
}
