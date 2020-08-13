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

func TestUpload(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessionId := vars["sessionId"]
	instanceName := vars["instanceName"]

  // Step 1: upload the code to the instance

	s, err := core.SessionGet(sessionId)
	if err == storage.NotFoundError {
		rw.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	i := core.InstanceGet(s, instanceName)

	// allow up to 32 MB which is the default

	red, err := req.MultipartReader()
	if err != nil {
		log.Println(err)
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	path := req.URL.Query().Get("path")

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

	// Step 1.5: Grab resources from server/endpoint

  // Grab the test name that we will use to match the directory at the host endpoint
  testName := req.URL.Query().Get("testname")

  // Get the cut length from the resource host endpoint. We need this for wget.
  // If the endpoint contains http or https, we ignore that.
  // e.g.: if the endpoint is https://localhost/llnl-freecompilercamp/freecc_tests/
  // the cutlength is 3.
  var cutlen = strings.Count(config.TestEndpoint, "/")
  if strings.Contains(config.TestEndpoint, "http") || strings.Contains(config.TestEndpoint, "https") {
    cutlen = cutlen - 2
  }

  // NOTE: The default WORKDIR set by the special rose-test and llvm-test images
  // is the directory where code is uploaded. All of the commands below are
  // executed in that directory implicitly (no need to cd)
  // cutlen + 1 because it includes the folder for this specific test.
  var wgetCmd = fmt.Sprintf(
    `{ "command":
    ["wget", "-r", "-np", "-R", "index.html*", "-nH", "--cut-dirs=%d",
    "%s/%s/"] }`,
    cutlen + 1, config.TestEndpoint, testName)

  var er1 execRequest // from exec.go handler

  err = json.NewDecoder(strings.NewReader(wgetCmd)).Decode(&er1)
  if err != nil {
    log.Fatal(err)
    rw.WriteHeader(http.StatusBadRequest)
    return
  }

  code, err := core.InstanceExec(i, er1.Cmd)

  if err != nil {
    log.Printf("Error executing command; status code: %s, error: %s", code, err)
    rw.WriteHeader(http.StatusInternalServerError)
    return
  }

  log.Printf("Obtained test resources from endpoint.")


  // Step 2: Compile via make

  var makeCmd = `{ "command": ["make", "-B"] }`

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

  // Parses the make response the word "Error" that shows when make fails.
  // If there is an error, send it back with the HTTP response. Else, send success.
  // NOTE: We send back a 502 (Bad Gateway) on compile error, along with the error(s).
  // Client should handle this appropriately (check if 502 and print error)
  if strings.Contains(buf.String(), "Error") {
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
