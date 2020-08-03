/*
* Specialized version of file_upload that is intended to upload code.
* It does the following:
*   - Uploads the code to the instance via the multipart form data
*   - Compiles the code
*   - Runs test cases on the code
*   - Once all are done, runs the test cases on the compiled code
*   - Returns an HTTP response once the above steps are completed, or the first
*       error encountered
*/

package handlers

import (
  "strings"
	"io"
	"log"
	"net/http"
  "encoding/json"

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

	// Step 2: compile the code
  const jsonStream =
  `
    { "command": ["gcc", "test.C", "-o", "test"] }
  `

  var er execRequest // from exec.go handler

  err = json.NewDecoder(strings.NewReader(jsonStream)).Decode(&er)
  if err != nil {
    log.Fatal(err)
    rw.WriteHeader(http.StatusBadRequest)
    return
  }

  cmdout, err := core.InstanceExecOutput(i, er.Cmd)

  if err != nil {
    log.Println(err)
    rw.WriteHeader(http.StatusInternalServerError)
    return
  }

  /*
  buf := new(strings.Builder)
  io.Copy(buf, cmdout)
  log.Println(buf.String())
  */

  rw.Header().Set("content-type", "application/json")
  if _, err = io.Copy(rw, cmdout); err != nil {
    log.Println(err)
    rw.WriteHeader(http.StatusInternalServerError)
    return
  }
  rw.WriteHeader(http.StatusOK)
  return
}
