package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"github.com/hornbill/color"
	_ "github.com/hornbill/go-mssqldb" //Microsoft SQL Server driver - v2005+
	"github.com/hornbill/goapiLib"
	_ "github.com/hornbill/mysql"    //MySQL v4.1 to v5.x and MariaDB driver
	_ "github.com/hornbill/mysql320" //MySQL v3.2.0 to v5 driver - Provides SWSQL (MySQL 4.0.16) support
	"github.com/hornbill/pb"
	"github.com/hornbill/sqlx"
	"html"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	version           = "1.2.9"
	appServiceManager = "com.hornbill.servicemanager"
	//Disk Space Declarations
	sizeKB float64 = 1 << (10 * 1)
	sizeMB float64 = 1 << (10 * 2)
	sizeGB float64 = 1 << (10 * 3)
	sizeTB float64 = 1 << (10 * 4)
	sizePB float64 = 1 << (10 * 5)
)

var (
	appDBDriver            string
	cacheDBDriver          string
	arrCallsLogged         = make(map[string]string)
	arrCallDetailsMaps     = make([]map[string]interface{}, 0)
	arrSWStatus            = make(map[string]string)
	boolConfLoaded         bool
	boolProcessClass       bool
	configFileName         string
	configZone             string
	configDryRun           bool
	configMaxRoutines      string
	connStrSysDB           string
	connStrAppDB           string
	counters               counterTypeStruct
	mapGenericConf         swCallConfStruct
	analysts               []analystListStruct
	categories             []categoryListStruct
	closeCategories        []categoryListStruct
	customers              []customerListStruct
	priorities             []priorityListStruct
	services               []serviceListStruct
	sites                  []siteListStruct
	teams                  []teamListStruct
	importFiles            []fileAssocStruct
	sqlCallQuery           string
	swImportConf           swImportConfStruct
	timeNow                string
	startTime              time.Time
	endTime                time.Duration
	espXmlmc               *apiLib.XmlmcInstStruct
	xmlmcInstanceConfig    xmlmcConfigStruct
	mutex                  = &sync.Mutex{}
	mutexAnalysts          = &sync.Mutex{}
	mutexArrCallsLogged    = &sync.Mutex{}
	mutexBar               = &sync.Mutex{}
	mutexCategories        = &sync.Mutex{}
	mutexCloseCategories   = &sync.Mutex{}
	mutexCustomers         = &sync.Mutex{}
	mutexPriorities        = &sync.Mutex{}
	mutexServices          = &sync.Mutex{}
	mutexSites             = &sync.Mutex{}
	mutexTeams             = &sync.Mutex{}
	wgRequest              sync.WaitGroup
	wgAssoc                sync.WaitGroup
	wgFile                 sync.WaitGroup
	reqPrefix              string
	maxGoroutines          = 1
	boolProcessAttachments bool
)

// ----- Structures -----
type counterTypeStruct struct {
	sync.Mutex
	created        int
	createdSkipped int
}

//----- Config Data Structs
type swImportConfStruct struct {
	HBConf                    hbConfStruct //Hornbill Instance connection details
	SWServerAddress           string
	AttachmentRoot            string
	CustomerType              string
	SMProfileCodeSeperator    string
	SWSystemDBConf            sysDBConfStruct //Cache Data (sw_systemdb) connection details
	SWAppDBConf               appDBConfStruct //App Data (swdata) connection details
	ConfIncident              swCallConfStruct
	ConfServiceRequest        swCallConfStruct
	ConfChangeRequest         swCallConfStruct
	ConfProblem               swCallConfStruct
	ConfKnownError            swCallConfStruct
	PriorityMapping           map[string]interface{}
	TeamMapping               map[string]interface{}
	CategoryMapping           map[string]interface{}
	ResolutionCategoryMapping map[string]interface{}
	ServiceMapping            map[string]interface{}
}
type hbConfStruct struct {
	UserName   string
	Password   string
	InstanceID string
	URL        string
}
type sysDBConfStruct struct {
	Driver   string
	UserName string
	Password string
}
type appDBConfStruct struct {
	Driver   string
	Server   string
	UserName string
	Password string
	Port     int
	Database string
	Encrypt  bool
}
type swCallConfStruct struct {
	Import                 bool
	CallClass              string
	DefaultTeam            string
	DefaultPriority        string
	DefaultService         string
	SQLStatement           string
	CoreFieldMapping       map[string]interface{}
	AdditionalFieldMapping map[string]interface{}
}

//----- XMLMC Config and Interaction Structs
type xmlmcConfigStruct struct {
	instance string
	url      string
	zone     string
}
type xmlmcResponse struct {
	MethodResult string      `xml:"status,attr"`
	State        stateStruct `xml:"state"`
}

//----- Shared Structs -----
type stateStruct struct {
	Code     string `xml:"code"`
	ErrorRet string `xml:"error"`
}

//----- Data Structs -----

type xmlmcSysSettingResponse struct {
	MethodResult string      `xml:"status,attr"`
	State        stateStruct `xml:"state"`
	Setting      string      `xml:"params>option>value"`
}

//----- Request Logged Structs
type xmlmcRequestResponseStruct struct {
	MethodResult string      `xml:"status,attr"`
	RequestID    string      `xml:"params>primaryEntityData>record>h_pk_reference"`
	SiteCountry  string      `xml:"params>rowData>row>h_country"`
	State        stateStruct `xml:"state"`
}
type xmlmcBPMSpawnedStruct struct {
	MethodResult string      `xml:"status,attr"`
	Identifier   string      `xml:"params>identifier"`
	State        stateStruct `xml:"state"`
}

//----- Site Structs
type siteListStruct struct {
	SiteName string
	SiteID   int
}
type xmlmcSiteListResponse struct {
	MethodResult string      `xml:"status,attr"`
	SiteID       int         `xml:"params>rowData>row>h_id"`
	SiteName     string      `xml:"params>rowData>row>h_site_name"`
	SiteCountry  string      `xml:"params>rowData>row>h_country"`
	State        stateStruct `xml:"state"`
}

//----- Priority Structs
type priorityListStruct struct {
	PriorityName string
	PriorityID   int
}
type xmlmcPriorityListResponse struct {
	MethodResult string      `xml:"status,attr"`
	PriorityID   int         `xml:"params>rowData>row>h_pk_priorityid"`
	PriorityName string      `xml:"params>rowData>row>h_priorityname"`
	State        stateStruct `xml:"state"`
}

//----- Service Structs
type serviceListStruct struct {
	ServiceName          string
	ServiceID            int
	ServiceBPMIncident   string
	ServiceBPMService    string
	ServiceBPMChange     string
	ServiceBPMProblem    string
	ServiceBPMKnownError string
}
type xmlmcServiceListResponse struct {
	MethodResult  string      `xml:"status,attr"`
	ServiceID     int         `xml:"params>rowData>row>h_pk_serviceid"`
	ServiceName   string      `xml:"params>rowData>row>h_servicename"`
	BPMIncident   string      `xml:"params>rowData>row>h_incident_bpm_name"`
	BPMService    string      `xml:"params>rowData>row>h_service_bpm_name"`
	BPMChange     string      `xml:"params>rowData>row>h_change_bpm_name"`
	BPMProblem    string      `xml:"params>rowData>row>h_problem_bpm_name"`
	BPMKnownError string      `xml:"params>rowData>row>h_knownerror_bpm_name"`
	State         stateStruct `xml:"state"`
}

//----- Team Structs
type teamListStruct struct {
	TeamName string
	TeamID   string
}
type xmlmcTeamListResponse struct {
	MethodResult string      `xml:"status,attr"`
	TeamID       string      `xml:"params>rowData>row>h_id"`
	TeamName     string      `xml:"params>rowData>row>h_name"`
	State        stateStruct `xml:"state"`
}

//----- Category Structs
type categoryListStruct struct {
	CategoryCode string
	CategoryID   string
	CategoryName string
}
type xmlmcCategoryListResponse struct {
	MethodResult string      `xml:"status,attr"`
	CategoryID   string      `xml:"params>id"`
	CategoryName string      `xml:"params>fullname"`
	State        stateStruct `xml:"state"`
}

//----- Audit Structs
type xmlmcAuditListResponse struct {
	MethodResult     string      `xml:"status,attr"`
	TotalStorage     float64     `xml:"params>maxStorageAvailble"`
	TotalStorageUsed float64     `xml:"params>totalStorageUsed"`
	State            stateStruct `xml:"state"`
}

//----- Analyst Structs
type analystListStruct struct {
	AnalystID   string
	AnalystName string
}
type xmlmcAnalystListResponse struct {
	MethodResult     string      `xml:"status,attr"`
	AnalystFullName  string      `xml:"params>name"`
	AnalystFirstName string      `xml:"params>firstName"`
	AnalystLastName  string      `xml:"params>lastName"`
	State            stateStruct `xml:"state"`
}

//----- Customer Structs
type customerListStruct struct {
	CustomerID   string
	CustomerName string
}
type xmlmcCustomerListResponse struct {
	MethodResult      string      `xml:"status,attr"`
	CustomerFirstName string      `xml:"params>firstName"`
	CustomerLastName  string      `xml:"params>lastName"`
	State             stateStruct `xml:"state"`
}

//----- Associated Record Struct
type reqRelStruct struct {
	MasterRef string `db:"fk_callref_m"`
	SlaveRef  string `db:"fk_callref_s"`
}

//----- File Attachment Structs
type xmlmcAttachmentResponse struct {
	MethodResult    string      `xml:"status,attr"`
	ContentLocation string      `xml:"params>contentLocation"`
	State           stateStruct `xml:"state"`
	HistFileID			string			`xml:"params>primaryEntityData>record>h_pk_fileid"`
}

//----- Email Attachment Structs
type xmlmcEmailAttachmentResponse struct {
	MethodResult string            `xml:"status,attr"`
	Recipients   []recipientStruct `xml:"params>recipient"`
	Subject      string            `xml:"params>subject"`
	Body         string            `xml:"params>body"`
	HTMLBody     string            `xml:"params>htmlBody"`
	TimeSent     string            `xml:"params>timeSent"`
	State        stateStruct       `xml:"state"`
}
type recipientStruct struct {
	Class   string `xml:"class"`
	Address string `xml:"address"`
	Name    string `xml:"name"`
}

//----- File Attachment Struct
type fileAssocStruct struct {
	ImportRef  int
	SmCallRef  string
	FileID     string  `db:"fileid"`
	CallRef    string  `db:"callref"`
	DataID     string  `db:"dataid"`
	UpdateID   string  `db:"updateid"`
	Compressed string  `db:"compressed"`
	SizeU      float64 `db:"sizeu"`
	SizeC      float64 `db:"sizec"`
	FileName   string  `db:"filename"`
	AddedBy    string  `db:"addedby"`
	TimeAdded  string  `db:"timeadded"`
	FileTime   string  `db:"filetime"`
}

// main package
func main() {
	//-- Start Time for Durration
	startTime = time.Now()
	//-- Start Time for Log File
	timeNow = time.Now().Format(time.RFC3339)
	timeNow = strings.Replace(timeNow, ":", "-", -1)

	arrSWStatus["1"] = "status.open"
	arrSWStatus["2"] = "status.open"
	arrSWStatus["3"] = "status.open"
	arrSWStatus["4"] = "status.onHold"
	arrSWStatus["5"] = "status.open"
	arrSWStatus["6"] = "status.resolved"
	arrSWStatus["8"] = "status.new"
	arrSWStatus["9"] = "status.open"
	arrSWStatus["10"] = "status.open"
	arrSWStatus["11"] = "status.open"
	arrSWStatus["16"] = "status.closed"
	arrSWStatus["17"] = "status.cancelled"
	arrSWStatus["18"] = "status.closed"

	//-- Grab and Parse Flags
	flag.StringVar(&configFileName, "file", "conf.json", "Name of the configuration file to load")
	flag.StringVar(&configZone, "zone", "eur", "Override the default Zone the instance sits in")
	flag.BoolVar(&configDryRun, "dryrun", false, "Dump import XML to log instead of creating requests")
	flag.StringVar(&configMaxRoutines, "concurrent", "1", "Maximum number of requests to import concurrently.")
	flag.BoolVar(&boolProcessAttachments, "attachments", false, "Import attachemnts without prompting.")
	flag.Parse()

	//-- Output to CLI and Log
	logger(1, "---- Supportworks Call Import Utility V"+fmt.Sprintf("%v", version)+" ----", true)
	logger(1, "Flag - Config File "+fmt.Sprintf("%s", configFileName), true)
	logger(1, "Flag - Zone "+fmt.Sprintf("%s", configZone), true)
	logger(1, "Flag - Dry Run "+fmt.Sprintf("%v", configDryRun), true)
	logger(1, "Flag - Concurrent Requests "+fmt.Sprintf("%v", configMaxRoutines), true)

	//Check maxGoroutines for valid value
	maxRoutines, err := strconv.Atoi(configMaxRoutines)
	if err != nil {
		color.Red("Unable to convert maximum concurrency of [" + configMaxRoutines + "] to type INT for processing")
		return
	}
	maxGoroutines = maxRoutines

	if maxGoroutines < 1 || maxGoroutines > 10 {
		color.Red("The maximum concurrent requests allowed is between 1 and 10 (inclusive).\n\n")
		color.Red("You have selected " + configMaxRoutines + ". Please try again, with a valid value against ")
		color.Red("the -concurrent switch.")
		return
	}

	//-- Load Configuration File Into Struct
	swImportConf, boolConfLoaded = loadConfig()
	if boolConfLoaded != true {
		logger(4, "Unable to load config, process closing.", true)
		return
	}

	//Set SQL driver ID string for Application Data
	if swImportConf.SWAppDBConf.Driver == "" {
		logger(4, "SWAppDBConf SQL Driver not set in configuration.", true)
		return
	}
	if swImportConf.SWAppDBConf.Driver == "swsql" {
		appDBDriver = "mysql320"
	} else if swImportConf.SWAppDBConf.Driver == "mysql" || swImportConf.SWAppDBConf.Driver == "mssql" || swImportConf.SWAppDBConf.Driver == "mysql320" {
		appDBDriver = swImportConf.SWAppDBConf.Driver
	} else {
		logger(4, "The SQL driver ("+swImportConf.SWAppDBConf.Driver+") for the Supportworks Application Database specified in the configuration file is not valid.", true)
		return
	}
	//Set SQL driver ID string for Cache Data
	if swImportConf.SWSystemDBConf.Driver == "" {
		logger(4, "SWSystemDBConf SQL Driver not set in configuration.", true)
		return
	}
	if swImportConf.SWSystemDBConf.Driver == "swsql" {
		cacheDBDriver = "mysql320"
	} else if swImportConf.SWSystemDBConf.Driver == "mysql" || swImportConf.SWSystemDBConf.Driver == "mysql320" {
		cacheDBDriver = swImportConf.SWSystemDBConf.Driver
	} else {
		logger(4, "The SQL driver ("+swImportConf.SWSystemDBConf.Driver+") for the Supportworks System Database specified in the configuration file is not valid.", true)
		return
	}

	//-- Set Instance ID
	SetInstance(configZone, swImportConf.HBConf.InstanceID)
	//-- Generate Instance XMLMC Endpoint
	swImportConf.HBConf.URL = getInstanceURL()

	//-- Log in to Hornbill instance
	var boolLogin = login()
	if boolLogin != true {
		logger(4, "Unable to Login ", true)
		return
	}
	//-- Defer log out of Hornbill instance until after main() is complete
	defer logout()

	//-- Build DB connection strings for sw_systemdb and swdata
	connStrSysDB = buildConnectionString("cache")
	connStrAppDB = buildConnectionString("app")

	//Process Incidents
	mapGenericConf = swImportConf.ConfIncident
	if mapGenericConf.Import == true {
		reqPrefix = getRequestPrefix("IN")
		processCallData()
	}
	//Process Service Requests
	mapGenericConf = swImportConf.ConfServiceRequest
	if mapGenericConf.Import == true {
		reqPrefix = getRequestPrefix("SR")
		processCallData()
	}
	//Process Change Requests
	mapGenericConf = swImportConf.ConfChangeRequest
	if mapGenericConf.Import == true {
		reqPrefix = getRequestPrefix("CH")
		processCallData()
	}
	//Process Problems
	mapGenericConf = swImportConf.ConfProblem
	if mapGenericConf.Import == true {
		reqPrefix = getRequestPrefix("PM")
		processCallData()
	}
	//Process Known Errors
	mapGenericConf = swImportConf.ConfKnownError
	if mapGenericConf.Import == true {
		reqPrefix = getRequestPrefix("KE")
		processCallData()
	}

	if len(arrCallsLogged) > 0 {
		//We have new calls logged - process associations
		processCallAssociations()
		//Process File Attachments
		processFileAttachments()
	}

	//-- End output
	logger(1, "Requests Logged: "+fmt.Sprintf("%d", counters.created), true)
	logger(1, "Requests Skipped: "+fmt.Sprintf("%d", counters.createdSkipped), true)
	//-- Show Time Takens
	endTime = time.Now().Sub(startTime)
	logger(1, "Time Taken: "+fmt.Sprintf("%v", endTime), true)
	logger(1, "---- Supportworks Call Import Complete ---- ", true)
}

//getRequestPrefix - gets and returns current maxResultsAllowed sys setting value
func getRequestPrefix(callclass string) string {
	espXmlmc, sessErr := NewEspXmlmcSession()
	if sessErr != nil {
		logger(4, "Unable to attach to XMLMC session to get Request Prefix. Using default ["+callclass+"].", false)
		return callclass
	}
	strSetting := ""
	switch callclass {
	case "IN":
		strSetting = "guest.app.requests.types.IN"
	case "SR":
		strSetting = "guest.app.requests.types.SR"
	case "CH":
		strSetting = "app.requests.types.CH"
	case "PM":
		strSetting = "app.requests.types.PM"
	case "KE":
		strSetting = "app.requests.types.KE"
	}

	espXmlmc.SetParam("appName", appServiceManager)
	espXmlmc.SetParam("filter", strSetting)
	response, err := espXmlmc.Invoke("admin", "appOptionGet")
	if err != nil {
		logger(4, "Could not retrieve System Setting for Request Prefix. Using default ["+callclass+"].", false)
		return callclass
	}
	var xmlRespon xmlmcSysSettingResponse
	err = xml.Unmarshal([]byte(response), &xmlRespon)
	if err != nil {
		logger(4, "Could not retrieve System Setting for Request Prefix. Using default ["+callclass+"].", false)
		return callclass
	}
	if xmlRespon.MethodResult != "ok" {
		logger(4, "Could not retrieve System Setting for Request Prefix: "+xmlRespon.MethodResult, false)
		return callclass
	}
	return xmlRespon.Setting
}

func processFileAttachments() {
	logger(1, "Processing Request File Attachments, please wait", true)
	//-- Build recurring map structure
	var callLevel = make(map[string]map[string]fileAssocStruct)

	//Connect to the JSON specified DB
	db, err := sqlx.Open(cacheDBDriver, connStrSysDB)
	defer db.Close()
	if err != nil {
		logger(4, " [DATABASE] Database Connection Error for Request File Attachments: "+fmt.Sprintf("%v", err), false)
		return
	}
	//Check connection is open
	err = db.Ping()
	if err != nil {
		logger(4, " [DATABASE] [PING] Database Connection Error for Request File Attachments: "+fmt.Sprintf("%v", err), false)
		return
	}
	logger(3, "[DATABASE] Connection Successful", false)
	logger(3, "[DATABASE] Running query for Request File Attachments. Please wait...", false)

	//build query
	sqlFileQuery := "SELECT fileid, callref, dataid, updateid, compressed, sizeu, sizec, filename, addedby, timeadded, filetime"
	sqlFileQuery = sqlFileQuery + " FROM system_cfastore "
	logger(3, "[DATABASE} Request File Attachments Query: "+sqlFileQuery, false)
	//Run Query
	rows, err := db.Queryx(sqlFileQuery)
	if err != nil {
		logger(4, " Database Query Error: "+fmt.Sprintf("%v", err), false)
		return
	}
	//-- Iterate through file attachment records returned from SQL query:
	// Where we have a corresponding request imported, insert file record in to recurring map for further processing

	for rows.Next() {
		//Scan current file attachment record in to struct
		var requestAttachments fileAssocStruct
		err = rows.StructScan(&requestAttachments)
		if err != nil {
			logger(4, " Data Mapping Error: "+fmt.Sprintf("%v", err), false)
			return
		}
		//Check to see if the file attachment matches a request that we've successfully imported
		currSMFileCallRef, importedCallHasAttachments := arrCallsLogged[requestAttachments.CallRef]
		if importedCallHasAttachments == true && currSMFileCallRef != "" {
			// Check if we already have an entry in the map for the current request
			// If not - create a map within the main map to hold the current file data
			_, callAlreadyMapped := callLevel[currSMFileCallRef]
			if callAlreadyMapped == false {
				callLevel[currSMFileCallRef] = make(map[string]fileAssocStruct)
			}
			requestAttachments.ImportRef = len(importFiles) + 1
			requestAttachments.SmCallRef = currSMFileCallRef
			importFiles = append(importFiles, requestAttachments)

			// Add an entry to the request map
			currDataID := requestAttachments.DataID
			callLevel[currSMFileCallRef][currDataID] = requestAttachments
		}
	}

	_, _, strStorageTotal, strStorageAvailable := getInstanceFreeSpace()
	var fltStorageRequired float64

	//Iterate through importFiles to get file attachment size information
	for _, fileRecord := range importFiles {
		fltStorageRequired = fltStorageRequired + fileRecord.SizeU
	}
	strStorageRequired := convFloattoSizeStr(fltStorageRequired)

	logger(6, "\n ------------ File Attachment Processing ------------", true)
	logger(6, " Approximately "+strStorageRequired+" of storage space is required to import your", true)
	logger(6, " Request File Attachments.", true)
	logger(6, " You have approximately "+strStorageAvailable+" available space, from your subscribed", true)
	logger(6, " amount of "+strStorageTotal+".", true)

	//check if we want to process attachments from command line flag otherwise ask
	if boolProcessAttachments == false {
		fmt.Printf(" Do you want to import your Supportworks Call File Attachments\n in to your Service Manager Requests (yes/no): ")
		if confirmResponse() == false {
			color.Red(" If you do not import file attachments at this stage, you will NOT\n be able to import them in the future!")
			color.Red("\n Please confirm your response one more time.")
			fmt.Printf("\n Do you want to import your Supportworks Call File Attachments\n in to your Service Manager Requests (yes/no): ")
			boolProcessAttachments = confirmResponse()
		} else {
			boolProcessAttachments = true
		}
	}
	if boolProcessAttachments == true {
		//Iterate through File Attachment records again for processing

		logger(3, " Processing "+fmt.Sprintf("%v", len(importFiles))+" attachments for "+fmt.Sprintf("%v", len(callLevel))+" requests...", true)
		bar := pb.StartNew(len(importFiles))
		maxGoroutinesGuard := make(chan struct{}, maxGoroutines)
		for _, fileRecord := range importFiles {

			maxGoroutinesGuard <- struct{}{}
			wgFile.Add(1)

			entityRequest := ""
			objFileRecord := fileRecord

			if objFileRecord.UpdateID == "999999999" {
				entityRequest = "Requests"
			} else {
				entityRequest = "RequestHistoricUpdateAttachments"
			}

			go func() {
				defer wgFile.Done()
				time.Sleep(150 * time.Millisecond)
				mutexBar.Lock()
				bar.Increment()
				mutexBar.Unlock()

				addFileContent(entityRequest, objFileRecord)

				<-maxGoroutinesGuard
			}()
		}
		wgFile.Wait()

		bar.FinishPrint("Request File Attachment Processing Complete")
		logger(1, "Request File Attachment Processing Complete", false)
	} else {
		logger(1, "No file attachments will be imported.", true)
	}
}

//decodeSWMFile - reads the email attachment from Supportworks, returns the content as a string
func decodeSWMFile(fileEncoded string) (string, string) {
	emailContent := ""
	subjectLine := ""
	//Decode SWM in to struct
	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		logger(4, "Unable to attach to XMLMC session to decode SWM.", true)
		return emailContent, subjectLine
	}
	espXmlmc.SetParam("fileContent", fileEncoded)
	XMLEmailDecoded, xmlmcErrEmail := espXmlmc.Invoke("mail", "decodeCompositeMessage")
	if xmlmcErrEmail != nil {
		logger(5, "API Error response from decodeCompositeMessage: "+fmt.Sprintf("%v", xmlmcErrEmail), false)
		return emailContent, subjectLine
	}
	var xmlResponEmail xmlmcEmailAttachmentResponse
	errUnmarshall := xml.Unmarshal([]byte(XMLEmailDecoded), &xmlResponEmail)
	if errUnmarshall != nil {
		logger(5, "Unable to read XML response from Message Decode: "+fmt.Sprintf("%v", errUnmarshall), false)
		return emailContent, subjectLine
	}
	if xmlResponEmail.MethodResult != "ok" {
		logger(5, "Error returned from API for Message Decode: "+fmt.Sprintf("%v", xmlResponEmail.MethodResult), false)
		return emailContent, subjectLine
	}

	if xmlResponEmail.Recipients == nil {
		logger(5, "No recipients found in mail message.", false)
		return emailContent, subjectLine
	}

	//Build string to write to text file
	fromAddress := ""
	toAddress := ""
	for _, recipient := range xmlResponEmail.Recipients {
		if recipient.Class == "from" {
			fromAddress = recipient.Address
		}
		if recipient.Class == "to" {
			toAddress = recipient.Address
		}
	}
	bodyText := ""
	if xmlResponEmail.Body != "" {
		bodyText = xmlResponEmail.Body
	} else {
		bodyText = xmlResponEmail.HTMLBody
	}
	subjectLine = "Subject: " + xmlResponEmail.Subject
	emailContent = "From: " + fromAddress + "\r\n"
	emailContent = emailContent + "To: " + toAddress + "\r\n"
	if xmlResponEmail.TimeSent != "" {
		emailContent = emailContent + "Sent: " + epochToDateTime(xmlResponEmail.TimeSent) + "\r\n"
	}
	emailContent = emailContent + subjectLine + "\r\n"
	emailContent = emailContent + strings.Repeat("-", len(subjectLine)) + "\r\n"
	emailContent = emailContent + bodyText
	return emailContent, subjectLine
}

//addFileContent - reads the file attachment from Supportworks, attach to request and update content location
func addFileContent(entityName string, fileRecord fileAssocStruct) bool {

	subFolderName := getSubFolderName(fileRecord.CallRef)
	hostFileName := padCallRef(fileRecord.CallRef, "f", 8) + "." + padCallRef(fileRecord.DataID, "", 3)
	fullFilePath := swImportConf.AttachmentRoot + "/" + subFolderName + "/" + hostFileName
	logger(1, "Adding file content from: "+fullFilePath, false)

	if _, fileCheckErr := os.Stat(fullFilePath); os.IsNotExist(fileCheckErr) {
		logger(4, "File does not exist at location.", false)
	}
	//-- Load Config File
	file, fileError := os.Open(fullFilePath)
	//-- Check For Error Reading File
	if fileError != nil {
		logger(4, "Error Opening File: "+fmt.Sprintf("%v", fileError), true)
		return false
	}
	defer file.Close()
	// create a new buffer base on file size
	fInfo, _ := file.Stat()
	var size int64
	size = fInfo.Size()
	buf := make([]byte, size)

	// read file content into buffer
	fReader := bufio.NewReader(file)
	fReader.Read(buf)
	fileEncoded := base64.StdEncoding.EncodeToString(buf)

	//If using the Requests entity, set primary key to be the SM request ref
	attPriKey := fileRecord.FileID
	if entityName == "Requests" {
		attPriKey = fileRecord.SmCallRef
	}

	fileExtension := filepath.Ext(fileRecord.FileName)
	swmDecoded := ""
	subjectLine := ""
	useFileName := fileRecord.FileName
	if fileExtension == ".swm" {
		//Further processing for SWM files
		//Copy content in to TXT file, and attach this instead
		swmDecoded, subjectLine = decodeSWMFile(fileEncoded)
		if swmDecoded != "" {
			fileEncoded = base64.StdEncoding.EncodeToString([]byte(swmDecoded))
		}
		useFileName = useFileName + ".txt"
	}

	filenameReplacer := strings.NewReplacer("<", "_", ">", "_", "|", "_", "\\", "_", "/", "_", ":", "_", "*", "_", "?", "_", "\"", "_")
	useFileName = fmt.Sprintf("%s", filenameReplacer.Replace(useFileName))

	if entityName == "RequestHistoricUpdateAttachments" {
		espXmlmc, sessErr := NewEspXmlmcSession()
		if sessErr != nil {
			logger(4, "Unable to attach to XMLMC session to add file record.", true)
			return false
		}
		espXmlmc.SetParam("application", appServiceManager)
		espXmlmc.SetParam("entity", "RequestHistoricUpdateAttachments")
		espXmlmc.SetParam("returnModifiedData", "true")
		espXmlmc.OpenElement("primaryEntityData")
		espXmlmc.OpenElement("record")
		espXmlmc.SetParam("h_addedby", fileRecord.AddedBy)
		espXmlmc.SetParam("h_callref", fileRecord.SmCallRef)
		espXmlmc.SetParam("h_compressed", fileRecord.Compressed)
		espXmlmc.SetParam("h_dataid", fileRecord.DataID)
		espXmlmc.SetParam("h_filename", useFileName)
		espXmlmc.SetParam("h_filetime", fileRecord.FileTime)
		espXmlmc.SetParam("h_pk_fileid", attPriKey)
		espXmlmc.SetParam("h_sizec", strconv.Itoa(int(fileRecord.SizeC)))
		espXmlmc.SetParam("h_sizeu", strconv.Itoa(int(fileRecord.SizeU)))
		espXmlmc.SetParam("h_timeadded", fileRecord.TimeAdded)
		espXmlmc.SetParam("h_updateid", fileRecord.UpdateID)
		espXmlmc.CloseElement("record")
		espXmlmc.CloseElement("primaryEntityData")

		var XMLSTRING = espXmlmc.GetParam()

		XMLHistAtt, xmlmcErr := espXmlmc.Invoke("data", "entityAddRecord")
		if xmlmcErr != nil {
			logger(1, "RequestHistoricUpdateAttachments entityAddRecord Failed "+fmt.Sprintf("%s", xmlmcErr), false)
			logger(1, "RequestHistoricUpdateAttachments entityAddRecord Failed File Attachment Record XML "+fmt.Sprintf("%s", XMLSTRING), false)
			return false
		}
		var xmlRespon xmlmcAttachmentResponse
		errXMLMC := xml.Unmarshal([]byte(XMLHistAtt), &xmlRespon)
		if errXMLMC != nil {
			logger(4, "Unable to read response from Hornbill instance for Update File Attachment Record Insertion ["+useFileName+"] ["+fileRecord.SmCallRef+"]:"+fmt.Sprintf("%v", errXMLMC), false)
			logger(1, "File Attachment Record XML "+fmt.Sprintf("%s", XMLSTRING), false)
			return false
		}
		if xmlRespon.MethodResult != "ok" {
			logger(4, "Unable to process Update File Attachment Record Insertion ["+useFileName+"] ["+fileRecord.SmCallRef+"]: "+xmlRespon.State.ErrorRet, false)
			logger(1, "File Attachment Record XML "+fmt.Sprintf("%s", XMLSTRING), false)
			return false
		}
		logger(1, "Historic Update File Attactment Record Insertion Success ["+useFileName+"] ["+fileRecord.SmCallRef+"]", false)
		attPriKey = xmlRespon.HistFileID
	}

	espXmlmc, sessErr2 := NewEspXmlmcSession()
	if sessErr2 != nil {
		logger(4, "Unable to attach to XMLMC session to add file record.", true)
		return false
	}
	//File content read - add data to instance
	espXmlmc.SetParam("application", appServiceManager)
	espXmlmc.SetParam("entity", entityName)
	espXmlmc.SetParam("keyValue", attPriKey)
	espXmlmc.SetParam("folder", "/")
	espXmlmc.OpenElement("localFile")
	espXmlmc.SetParam("fileName", useFileName)
	espXmlmc.SetParam("fileData", fileEncoded)
	espXmlmc.CloseElement("localFile")
	espXmlmc.SetParam("overwrite", "true")
	var XMLSTRINGDATA = espXmlmc.GetParam()
	XMLAttach, xmlmcErr := espXmlmc.Invoke("data", "entityAttachFile")
	if xmlmcErr != nil {
		logger(4, "Could not add Attachment File Data for ["+useFileName+"] ["+fileRecord.SmCallRef+"]: "+fmt.Sprintf("%v", xmlmcErr), false)
		logger(1, "File Data Record XML "+fmt.Sprintf("%s", XMLSTRINGDATA), false)
		return false
	}
	var xmlRespon xmlmcAttachmentResponse

	err := xml.Unmarshal([]byte(XMLAttach), &xmlRespon)
	if err != nil {
		logger(4, "Could not add Attachment File Data for ["+useFileName+"] ["+fileRecord.SmCallRef+"]: "+fmt.Sprintf("%v", err), false)
		logger(1, "File Data Record XML "+fmt.Sprintf("%s", XMLSTRINGDATA), false)
	} else {
		if xmlRespon.MethodResult != "ok" {
			logger(4, "Could not add Attachment File Data for ["+useFileName+"] ["+fileRecord.SmCallRef+"]: "+xmlRespon.State.ErrorRet, false)
			logger(1, "File Data Record XML "+fmt.Sprintf("%s", XMLSTRINGDATA), false)
		} else {
			//-- If we've got a Content Location back from the API, update the file record with this
			if xmlRespon.ContentLocation != "" {
				strService := ""
				strMethod := ""
				espXmlmc, sessErr3 := NewEspXmlmcSession()
				if sessErr3 != nil {
					logger(4, "Unable to attach to XMLMC session to add file record.", true)
					return false
				}
				if entityName == "RequestHistoricUpdateAttachments" {
					espXmlmc.SetParam("application", appServiceManager)
					espXmlmc.SetParam("entity", "RequestHistoricUpdateAttachments")
					espXmlmc.OpenElement("primaryEntityData")
					espXmlmc.OpenElement("record")
					espXmlmc.SetParam("h_pk_fileid", attPriKey)
					espXmlmc.SetParam("h_contentlocation", xmlRespon.ContentLocation)
					espXmlmc.CloseElement("record")
					espXmlmc.CloseElement("primaryEntityData")
					strService = "data"
					strMethod = "entityUpdateRecord"
				} else {
					espXmlmc.SetParam("application", appServiceManager)
					espXmlmc.SetParam("entity", "RequestAttachments")
					espXmlmc.OpenElement("primaryEntityData")
					espXmlmc.OpenElement("record")
					espXmlmc.SetParam("h_request_id", fileRecord.SmCallRef)
					if subjectLine != "" {
						espXmlmc.SetParam("h_description", subjectLine+" - Originally added by "+fileRecord.AddedBy)
					} else {
						espXmlmc.SetParam("h_description", "Originally added by "+fileRecord.AddedBy)
					}
					espXmlmc.SetParam("h_filename", useFileName)
					espXmlmc.SetParam("h_contentlocation", xmlRespon.ContentLocation)
					espXmlmc.SetParam("h_timestamp", fileRecord.TimeAdded)
					espXmlmc.SetParam("h_visibility", "trustedGuest")
					espXmlmc.CloseElement("record")
					espXmlmc.CloseElement("primaryEntityData")
					strService = "data"
					strMethod = "entityAddRecord"
				}
				XMLSTRINGDATA = espXmlmc.GetParam()
				XMLContentLoc, xmlmcErrContent := espXmlmc.Invoke(strService, strMethod)
				if xmlmcErrContent != nil {
					logger(4, "Could not update request ["+fileRecord.SmCallRef+"] with attachment ["+useFileName+"]: "+fmt.Sprintf("%v", xmlmcErrContent), false)
					logger(1, "File Data Record XML "+fmt.Sprintf("%s", XMLSTRINGDATA), false)
					return false
				}
				var xmlResponLoc xmlmcResponse

				err := xml.Unmarshal([]byte(XMLContentLoc), &xmlResponLoc)
				if err != nil {
					logger(4, "Added file data to but unable to set Content Location on ["+fileRecord.SmCallRef+"] for File Content ["+useFileName+"] - read response from Hornbill instance:"+fmt.Sprintf("%v", err), false)
					logger(1, "File Data Record XML "+fmt.Sprintf("%s", XMLSTRINGDATA), false)
					return false
				}
				if xmlResponLoc.MethodResult != "ok" {
					logger(4, "Added file data but unable to set Content Location on ["+fileRecord.SmCallRef+"] for File Content ["+useFileName+"]: "+xmlResponLoc.State.ErrorRet, false)
					logger(1, "File Data Record XML "+fmt.Sprintf("%s", XMLSTRINGDATA), false)
					return false
				}
				logger(1, entityName+" File Content ["+useFileName+"] Added to ["+fileRecord.SmCallRef+"] Successfully", false)
			}
		}
	}
	return true
}

//getSubFolderName - takes SW call reference, passes back the folder name where the calls attachments are stored
func getSubFolderName(fileCallRef string) string {
	paddedRef := padCallRef(fileCallRef, "", 7)
	folderName := ""
	for i := 0; i < 4; i++ {
		folderName = folderName + string(paddedRef[i])
	}
	return folderName
}

//confirmResponse - prompts user, expects a fuzzy yes or no response, does not continue until this is given
func confirmResponse() bool {
	var cmdResponse string
	_, errResponse := fmt.Scanln(&cmdResponse)
	if errResponse != nil {
		log.Fatal(errResponse)
	}
	if cmdResponse == "y" || cmdResponse == "yes" || cmdResponse == "Y" || cmdResponse == "Yes" || cmdResponse == "YES" {
		return true
	} else if cmdResponse == "n" || cmdResponse == "no" || cmdResponse == "N" || cmdResponse == "No" || cmdResponse == "NO" {
		return false
	} else {
		color.Red("Please enter yes or no to continue:")
		return confirmResponse()
	}
}

//convFloattoSizeStr - takes given float64 value, returns a human readable storage capacity string
func convFloattoSizeStr(floatNum float64) (strReturn string) {
	if floatNum >= sizePB {
		strReturn = fmt.Sprintf("%.2fPB", floatNum/sizePB)
	} else if floatNum >= sizeTB {
		strReturn = fmt.Sprintf("%.2fTB", floatNum/sizeTB)
	} else if floatNum >= sizeGB {
		strReturn = fmt.Sprintf("%.2fGB", floatNum/sizeGB)
	} else if floatNum >= sizeMB {
		strReturn = fmt.Sprintf("%.2fMB", floatNum/sizeMB)
	} else if floatNum >= sizeKB {
		strReturn = fmt.Sprintf("%.2fKB", floatNum/sizeKB)
	} else {
		strReturn = fmt.Sprintf("%vB", int(floatNum))
	}
	return
}

//getInstanceFreeSpace - calculates how much storage is available on the given Hornbill instance
func getInstanceFreeSpace() (int64, int64, string, string) {
	var fltTotalSpace float64
	var fltFreeSpace float64
	var strTotalSpace string
	var strFreeSpace string

	XMLAudit, xmlmcErr := espXmlmc.Invoke("admin", "getInstanceAuditInfo")
	if xmlmcErr != nil {
		logger(4, "Could not return Instance Audit Information: "+fmt.Sprintf("%v", xmlmcErr), true)
		return 0, 0, "0B", "0B"
	}
	var xmlRespon xmlmcAuditListResponse

	err := xml.Unmarshal([]byte(XMLAudit), &xmlRespon)
	if err != nil {
		logger(4, "Could not return Instance Audit Information: "+fmt.Sprintf("%v", err), true)
	} else {
		if xmlRespon.MethodResult != "ok" {
			logger(4, "Could not return Instance Audit Information: "+xmlRespon.State.ErrorRet, true)
		} else {
			//-- Check Response
			if xmlRespon.TotalStorage > 0 && xmlRespon.TotalStorageUsed > 0 {
				fltTotalSpace = xmlRespon.TotalStorage
				fltFreeSpace = xmlRespon.TotalStorage - xmlRespon.TotalStorageUsed
				strTotalSpace = convFloattoSizeStr(fltTotalSpace)
				strFreeSpace = convFloattoSizeStr(fltFreeSpace)
			}
		}
	}
	return int64(fltTotalSpace), int64(fltFreeSpace), strTotalSpace, strFreeSpace
}

//processCallAssociations - Get all records from swdata.cmn_rel_opencall_oc, process accordingly
func processCallAssociations() {
	logger(1, "Processing Request Associations, please wait...", true)
	//Connect to the JSON specified DB
	db, err := sqlx.Open(appDBDriver, connStrAppDB)
	defer db.Close()
	if err != nil {
		logger(4, " [DATABASE] Database Connection Error for Request Associations: "+fmt.Sprintf("%v", err), false)
		return
	}
	//Check connection is open
	err = db.Ping()
	if err != nil {
		logger(4, " [DATABASE] [PING] Database Connection Error for Request Associations: "+fmt.Sprintf("%v", err), false)
		return
	}
	logger(3, "[DATABASE] Connection Successful", false)
	logger(3, "[DATABASE] Running query for Request Associations. Please wait...", false)

	//build query
	sqlDiaryQuery := "SELECT fk_callref_m, fk_callref_s from cmn_rel_opencall_oc "
	logger(3, "[DATABASE] Request Association Query: "+sqlDiaryQuery, false)
	//Run Query
	rows, err := db.Queryx(sqlDiaryQuery)
	if err != nil {
		logger(4, " Database Query Error: "+fmt.Sprintf("%v", err), false)
		return
	}
	//Process each association record, insert in to Hornbill
	//fmt.Println("Maximum Request Association Go Routines:", maxGoroutines)
	maxGoroutinesGuard := make(chan struct{}, maxGoroutines)
	for rows.Next() {
		var requestRels reqRelStruct

		errDataMap := rows.StructScan(&requestRels)
		if errDataMap != nil {
			logger(4, " Data Mapping Error: "+fmt.Sprintf("%v", errDataMap), false)
			return
		}
		smMasterRef, mrOK := arrCallsLogged[requestRels.MasterRef]
		smSlaveRef, srOK := arrCallsLogged[requestRels.SlaveRef]
		maxGoroutinesGuard <- struct{}{}
		wgAssoc.Add(1)
		go func() {
			defer wgAssoc.Done()
			if mrOK == true && smMasterRef != "" && srOK == true && smSlaveRef != "" {
				//We have Master and Slave calls matched in the SM database
				addAssocRecord(smMasterRef, smSlaveRef)
			}
			<-maxGoroutinesGuard
		}()
	}
	wgAssoc.Wait()
	logger(1, "Request Association Processing Complete", true)
}

//addAssocRecord - given a Master Reference and a Slave Refernce, adds a call association record to Service Manager
func addAssocRecord(masterRef, slaveRef string) {
	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return
	}
	espXmlmc.SetParam("application", appServiceManager)
	espXmlmc.SetParam("entity", "RelatedRequests")
	espXmlmc.OpenElement("primaryEntityData")
	espXmlmc.OpenElement("record")
	espXmlmc.SetParam("h_fk_parentrequestid", masterRef)
	espXmlmc.SetParam("h_fk_childrequestid", slaveRef)
	espXmlmc.CloseElement("record")
	espXmlmc.CloseElement("primaryEntityData")
	XMLUpdate, xmlmcErr := espXmlmc.Invoke("data", "entityAddRecord")
	if xmlmcErr != nil {
		//		log.Fatal(xmlmcErr)
		logger(4, "Unable to create Request Association between ["+masterRef+"] and ["+slaveRef+"] :"+fmt.Sprintf("%v", xmlmcErr), false)
		return
	}
	var xmlRespon xmlmcResponse
	errXMLMC := xml.Unmarshal([]byte(XMLUpdate), &xmlRespon)
	if errXMLMC != nil {
		logger(4, "Unable to read response from Hornbill instance for Request Association between ["+masterRef+"] and ["+slaveRef+"] :"+fmt.Sprintf("%v", errXMLMC), false)
		return
	}
	if xmlRespon.MethodResult != "ok" {
		logger(3, "Unable to add Request Association between ["+masterRef+"] and ["+slaveRef+"] : "+xmlRespon.State.ErrorRet, false)
		return
	}
	logger(1, "Request Association Success between ["+masterRef+"] and ["+slaveRef+"]", false)
}

//processCallData - Query Supportworks call data, process accordingly
func processCallData() {
	if queryDBCallDetails(mapGenericConf.CallClass, connStrAppDB) == true {
		bar := pb.StartNew(len(arrCallDetailsMaps))
		//We have Call Details - insert them in to
		//fmt.Println("Maximum Request Go Routines:", maxGoroutines)
		maxGoroutinesGuard := make(chan struct{}, maxGoroutines)
		for _, callRecord := range arrCallDetailsMaps {
			maxGoroutinesGuard <- struct{}{}
			wgRequest.Add(1)
			callRecordArr := callRecord
			callRecordCallref := callRecord["callref"]

			go func() {
				defer wgRequest.Done()
				time.Sleep(1 * time.Millisecond)
				mutexBar.Lock()
				bar.Increment()
				mutexBar.Unlock()
				//callID := fmt.Sprintf("%s", callRecordCallref)
				callID := ""
				if callInt, ok := callRecordCallref.(int64); ok {
					callID = strconv.FormatInt(callInt, 10)
				} else {
					callID = fmt.Sprintf("%s", callRecordCallref)
				}

				currentCallRef := padCallRef(callID, "F", 7)

				boolCallLogged, hbCallRef := logNewCall(mapGenericConf.CallClass, callRecordArr, callID)
				if boolCallLogged {
					logger(3, "[REQUEST LOGGED] Request logged successfully: "+hbCallRef+" from Supportworks call "+currentCallRef, false)
				} else {
					logger(4, mapGenericConf.CallClass+" call log failed: "+currentCallRef, false)
				}
				<-maxGoroutinesGuard
			}()
		}
		wgRequest.Wait()

		bar.FinishPrint(mapGenericConf.CallClass + " Call Import Complete")
	} else {
		logger(4, "Call Search Failed for Call Class: "+mapGenericConf.CallClass, true)
	}
}

//queryDBCallDetails -- Query call data & set map of calls to add to Hornbill
func queryDBCallDetails(callClass, connString string) bool {
	if callClass == "" || connString == "" {
		return false
	}
	//Connect to the JSON specified DB
	db, err := sqlx.Open(appDBDriver, connString)
	defer db.Close()
	if err != nil {
		logger(4, " [DATABASE] Database Connection Error: "+fmt.Sprintf("%v", err), true)
		return false
	}
	//Check connection is open
	err = db.Ping()
	if err != nil {
		logger(4, " [DATABASE] [PING] Database Connection Error: "+fmt.Sprintf("%v", err), true)
		return false
	}
	logger(3, "[DATABASE] Connection Successful", true)
	logger(3, "[DATABASE] Running query for calls of class "+callClass+". Please wait...", true)

	//build query
	sqlCallQuery = mapGenericConf.SQLStatement
	logger(3, "[DATABASE] Query to retrieve "+callClass+" calls from Supportworks: "+sqlCallQuery, false)

	//Run Query
	rows, err := db.Queryx(sqlCallQuery)
	if err != nil {
		logger(4, " Database Query Error: "+fmt.Sprintf("%v", err), true)
		return false
	}
	//Clear down existing Call Details map
	arrCallDetailsMaps = nil
	//Build map full of calls to import
	intCallCount := 0
	for rows.Next() {
		intCallCount++
		results := make(map[string]interface{})
		err = rows.MapScan(results)
		//Stick marshalled data map in to parent slice
		arrCallDetailsMaps = append(arrCallDetailsMaps, results)
	}
	defer rows.Close()
	return true
}

//logNewCall - Function takes Supportworks call data in a map, and logs to Hornbill
func logNewCall(callClass string, callMap map[string]interface{}, swCallID string) (bool, string) {

	boolCallLoggedOK := false
	strNewCallRef := ""

	strStatus := ""
	statusMapping := fmt.Sprintf("%v", mapGenericConf.CoreFieldMapping["h_status"])
	if statusMapping != "" {
		if statusMapping == "16" || statusMapping == "18" {
			strStatus = arrSWStatus["6"]
		} else {
			strStatus = arrSWStatus[getFieldValue(statusMapping, callMap)]
		}
	}

	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return false, ""
	}

	espXmlmc.SetParam("application", appServiceManager)
	espXmlmc.SetParam("entity", "Requests")
	espXmlmc.SetParam("returnModifiedData", "true")
	espXmlmc.OpenElement("primaryEntityData")
	espXmlmc.OpenElement("record")
	strAttribute := ""
	strMapping := ""
	strServiceBPM := ""
	boolUpdateLogDate := false
	strLoggedDate := ""
	strClosedDate := ""
	//Loop through core fields from config, add to XMLMC Params
	for k, v := range mapGenericConf.CoreFieldMapping {
		boolAutoProcess := true
		strAttribute = fmt.Sprintf("%v", k)
		strMapping = fmt.Sprintf("%v", v)

		//Owning Analyst Name
		if strAttribute == "h_ownerid" {
			strOwnerID := getFieldValue(strMapping, callMap)
			if strOwnerID != "" {
				boolAnalystExists := doesAnalystExist(strOwnerID)
				if boolAnalystExists {
					//Get analyst from cache as exists
					analystIsInCache, strOwnerName := recordInCache(strOwnerID, "Analyst")
					if analystIsInCache && strOwnerName != "" {
						espXmlmc.SetParam(strAttribute, strOwnerID)
						espXmlmc.SetParam("h_ownername", strOwnerName)
					}
				}
			}
			boolAutoProcess = false
		}

		//Customer ID & Name
		if strAttribute == "h_fk_user_id" {
			strCustID := getFieldValue(strMapping, callMap)
			if strCustID != "" {
				boolCustExists := doesCustomerExist(strCustID)
				if boolCustExists {
					//Get customer from cache as exists
					customerIsInCache, strCustName := recordInCache(strCustID, "Customer")
					if customerIsInCache && strCustName != "" {
						espXmlmc.SetParam(strAttribute, strCustID)
						espXmlmc.SetParam("h_fk_user_name", strCustName)
					}
				}
			}
			boolAutoProcess = false
		}

		//Priority ID & Name
		//-- Get Priority ID
		if strAttribute == "h_fk_priorityid" {
			strPriorityID := getFieldValue(strMapping, callMap)
			strPriorityMapped, strPriorityName := getCallPriorityID(strPriorityID)
			if strPriorityMapped == "" && mapGenericConf.DefaultPriority != "" {
				strPriorityID = getPriorityID(mapGenericConf.DefaultPriority)
				strPriorityName = mapGenericConf.DefaultPriority
			}
			espXmlmc.SetParam(strAttribute, strPriorityMapped)
			espXmlmc.SetParam("h_fk_priorityname", strPriorityName)
			boolAutoProcess = false
		}

		// Category ID & Name
		if strAttribute == "h_category_id" && strMapping != "" {
			//-- Get Call Category ID
			strCategoryID, strCategoryName := getCallCategoryID(callMap, "Request")
			if strCategoryID != "" && strCategoryName != "" {
				espXmlmc.SetParam(strAttribute, strCategoryID)
				espXmlmc.SetParam("h_category", strCategoryName)
			}
			boolAutoProcess = false
		}

		// Closure Category ID & Name
		if strAttribute == "h_closure_category_id" && strMapping != "" {
			strClosureCategoryID, strClosureCategoryName := getCallCategoryID(callMap, "Closure")
			if strClosureCategoryID != "" {
				espXmlmc.SetParam(strAttribute, strClosureCategoryID)
				espXmlmc.SetParam("h_closure_category", strClosureCategoryName)
			}
			boolAutoProcess = false
		}

		// Service ID & Name, & BPM Workflow
		if strAttribute == "h_fk_serviceid" {
			//-- Get Service ID
			swServiceID := getFieldValue(strMapping, callMap)
			strServiceID := getCallServiceID(swServiceID)
			if strServiceID == "" && mapGenericConf.DefaultService != "" {
				strServiceID = getServiceID(mapGenericConf.DefaultService)
			}
			if strServiceID != "" {
				//-- Get record from Service Cache
				strServiceName := ""
				mutexServices.Lock()
				for _, service := range services {
					if strconv.Itoa(service.ServiceID) == strServiceID {
						strServiceName = service.ServiceName
						switch callClass {
						case "Incident":
							strServiceBPM = service.ServiceBPMIncident
						case "Service Request":
							strServiceBPM = service.ServiceBPMService
						case "Change Request":
							strServiceBPM = service.ServiceBPMChange
						case "Problem":
							strServiceBPM = service.ServiceBPMProblem
						case "Known Error":
							strServiceBPM = service.ServiceBPMKnownError
						}
					}
				}
				mutexServices.Unlock()

				if strServiceName != "" {
					espXmlmc.SetParam(strAttribute, strServiceID)
					espXmlmc.SetParam("h_fk_servicename", strServiceName)
				}
			}
			boolAutoProcess = false
		}

		// Request Status
		if strAttribute == "h_status" {
			espXmlmc.SetParam(strAttribute, strStatus)
			boolAutoProcess = false
		}

		// Team ID and Name
		if strAttribute == "h_fk_team_id" {
			//-- Get Team ID
			swTeamID := getFieldValue(strMapping, callMap)
			strTeamID, strTeamName := getCallTeamID(swTeamID)
			if strTeamID == "" && mapGenericConf.DefaultTeam != "" {
				strTeamName = mapGenericConf.DefaultTeam
				strTeamID = getTeamID(strTeamName)
			}
			if strTeamID != "" && strTeamName != "" {
				espXmlmc.SetParam(strAttribute, strTeamID)
				espXmlmc.SetParam("h_fk_team_name", strTeamName)
			}
			boolAutoProcess = false
		}

		// Site ID and Name
		if strAttribute == "h_site_id" {
			//-- Get site ID
			siteID, siteName := getSiteID(callMap)
			if siteID != "" && siteName != "" {
				espXmlmc.SetParam(strAttribute, siteID)
				espXmlmc.SetParam("h_site", siteName)
			}
			boolAutoProcess = false
		}

		// Resolved Date/Time
		if strAttribute == "h_dateresolved" && strMapping != "" && (strStatus == "status.resolved" || strStatus == "status.closed") {
			resolvedEPOCH := getFieldValue(strMapping, callMap)
			if resolvedEPOCH != "" && resolvedEPOCH != "0" {
				strResolvedDate := epochToDateTime(resolvedEPOCH)
				if strResolvedDate != "" {
					espXmlmc.SetParam(strAttribute, strResolvedDate)
				}
			}
		}

		// Closed Date/Time
		if strAttribute == "h_dateclosed" && strMapping != "" && (strStatus == "status.resolved" || strStatus == "status.closed" || strStatus == "status.onHold") {
			closedEPOCH := getFieldValue(strMapping, callMap)
			if closedEPOCH != "" && closedEPOCH != "0" {
				strClosedDate = epochToDateTime(closedEPOCH)
				if strClosedDate != "" && strStatus != "status.onHold" {
					espXmlmc.SetParam(strAttribute, strClosedDate)
				}
			}
		}

		// Log Date/Time - setup ready to be processed after call logged
		if strAttribute == "h_datelogged" && strMapping != "" {
			loggedEPOCH := getFieldValue(strMapping, callMap)
			if loggedEPOCH != "" && loggedEPOCH != "0" {
				strLoggedDate = epochToDateTime(loggedEPOCH)
				if strLoggedDate != "" {
					boolUpdateLogDate = true
				}
			}
		}

		//Everything Else
		if boolAutoProcess &&
			strAttribute != "h_requesttype" &&
			strAttribute != "h_request_prefix" &&
			strAttribute != "h_category" &&
			strAttribute != "h_closure_category" &&
			strAttribute != "h_fk_servicename" &&
			strAttribute != "h_fk_team_name" &&
			strAttribute != "h_site" &&
			strAttribute != "h_fk_priorityname" &&
			strAttribute != "h_ownername" &&
			strAttribute != "h_fk_user_name" &&
			strAttribute != "h_datelogged" &&
			strAttribute != "h_dateresolved" &&
			strAttribute != "h_dateclosed" {

			if strMapping != "" && getFieldValue(strMapping, callMap) != "" {
				espXmlmc.SetParam(strAttribute, getFieldValue(strMapping, callMap))
			}
		}

	}

	//Add request class & prefix
	espXmlmc.SetParam("h_requesttype", callClass)
	espXmlmc.SetParam("h_request_prefix", reqPrefix)

	espXmlmc.CloseElement("record")
	espXmlmc.CloseElement("primaryEntityData")

	//Class Specific Data Insert
	espXmlmc.OpenElement("relatedEntityData")
	espXmlmc.SetParam("relationshipName", "Call Type")
	espXmlmc.SetParam("entityAction", "insert")
	espXmlmc.OpenElement("record")
	strAttribute = ""
	strMapping = ""
	//Loop through AdditionalFieldMapping fields from config, add to XMLMC Params if not empty
	for k, v := range mapGenericConf.AdditionalFieldMapping {
		strAttribute = fmt.Sprintf("%v", k)
		strMapping = fmt.Sprintf("%v", v)
		if strMapping != "" && getFieldValue(strMapping, callMap) != "" {
			espXmlmc.SetParam(strAttribute, getFieldValue(strMapping, callMap))
		}
	}

	espXmlmc.CloseElement("record")
	espXmlmc.CloseElement("relatedEntityData")

	//-- Check for Dry Run
	if configDryRun != true {

		XMLCreate, xmlmcErr := espXmlmc.Invoke("data", "entityAddRecord")
		if xmlmcErr != nil {
			//log.Fatal(xmlmcErr)
			logger(4, "Unable to log request on Hornbill instance:"+fmt.Sprintf("%v", xmlmcErr), false)
			return false, "No"
		}
		var xmlRespon xmlmcRequestResponseStruct

		err := xml.Unmarshal([]byte(XMLCreate), &xmlRespon)
		if err != nil {
			counters.Lock()
			counters.createdSkipped++
			counters.Unlock()
			logger(4, "Unable to read response from Hornbill instance:"+fmt.Sprintf("%v", err), false)
			return false, "No"
		}
		if xmlRespon.MethodResult != "ok" {
			logger(4, "Unable to log request: "+xmlRespon.State.ErrorRet, false)
			counters.Lock()
			counters.createdSkipped++
			counters.Unlock()
		} else {
			strNewCallRef = xmlRespon.RequestID

			mutexArrCallsLogged.Lock()
			arrCallsLogged[swCallID] = strNewCallRef
			mutexArrCallsLogged.Unlock()

			counters.Lock()
			counters.created++
			counters.Unlock()
			boolCallLoggedOK = true

			//Now update the request to create the activity stream
			espXmlmc.SetParam("socialObjectRef", "urn:sys:entity:"+appServiceManager+":Requests:"+strNewCallRef)
			espXmlmc.SetParam("content", "Request imported from Supportworks")
			espXmlmc.SetParam("visibility", "public")
			espXmlmc.SetParam("type", "Logged")
			fixed, err := espXmlmc.Invoke("activity", "postMessage")
			if err != nil {
				logger(5, "Activity Stream Creation failed for Request: "+strNewCallRef, false)
			} else {
				var xmlRespon xmlmcResponse
				err = xml.Unmarshal([]byte(fixed), &xmlRespon)
				if err != nil {
					logger(5, "Activity Stream Creation unmarshall failed for Request "+strNewCallRef, false)
				} else {
					if xmlRespon.MethodResult != "ok" {
						logger(5, "Activity Stream Creation was unsuccessful for ["+strNewCallRef+"]: "+xmlRespon.MethodResult, false)
					} else {
						logger(1, "Activity Stream Creation successful for ["+strNewCallRef+"]", false)
					}
				}
			}

			//Now update Logdate
			if boolUpdateLogDate {
				espXmlmc.SetParam("application", appServiceManager)
				espXmlmc.SetParam("entity", "Requests")
				espXmlmc.OpenElement("primaryEntityData")
				espXmlmc.OpenElement("record")
				espXmlmc.SetParam("h_pk_reference", strNewCallRef)
				espXmlmc.SetParam("h_datelogged", strLoggedDate)
				espXmlmc.CloseElement("record")
				espXmlmc.CloseElement("primaryEntityData")
				XMLBPM, xmlmcErr := espXmlmc.Invoke("data", "entityUpdateRecord")
				if xmlmcErr != nil {
					//log.Fatal(xmlmcErr)
					logger(4, "Unable to update Log Date of request ["+strNewCallRef+"] : "+fmt.Sprintf("%v", xmlmcErr), false)
				}
				var xmlRespon xmlmcResponse

				errLogDate := xml.Unmarshal([]byte(XMLBPM), &xmlRespon)
				if errLogDate != nil {
					logger(4, "Unable to update Log Date of request ["+strNewCallRef+"] : "+fmt.Sprintf("%v", errLogDate), false)
				}
				if xmlRespon.MethodResult != "ok" {
					logger(4, "Unable to update Log Date of request ["+strNewCallRef+"] : "+xmlRespon.State.ErrorRet, false)
				}
			}

			//Now do BPM Processing
			if strStatus != "status.resolved" &&
				strStatus != "status.closed" &&
				strStatus != "status.cancelled" {

				logger(1, callClass+" Logged: "+strNewCallRef+". Open Request status, spawing BPM Process "+strServiceBPM, false)
				if strNewCallRef != "" && strServiceBPM != "" {
					espXmlmc.SetParam("application", appServiceManager)
					espXmlmc.SetParam("name", strServiceBPM)
					espXmlmc.OpenElement("inputParams")
					espXmlmc.SetParam("objectRefUrn", "urn:sys:entity:"+appServiceManager+":Requests:"+strNewCallRef)
					espXmlmc.SetParam("requestId", strNewCallRef)
					espXmlmc.CloseElement("inputParams")

					XMLBPM, xmlmcErr := espXmlmc.Invoke("bpm", "processSpawn")
					if xmlmcErr != nil {
						//log.Fatal(xmlmcErr)
						logger(4, "Unable to invoke BPM for request ["+strNewCallRef+"]: "+fmt.Sprintf("%v", xmlmcErr), false)
					}
					var xmlRespon xmlmcBPMSpawnedStruct

					errBPM := xml.Unmarshal([]byte(XMLBPM), &xmlRespon)
					if errBPM != nil {
						logger(4, "Unable to read response from Hornbill instance:"+fmt.Sprintf("%v", errBPM), false)
						return false, "No"
					}
					if xmlRespon.MethodResult != "ok" {
						logger(4, "Unable to invoke BPM: "+xmlRespon.State.ErrorRet, false)
					} else {
						//Now, associate spawned BPM to the new Request
						espXmlmc.SetParam("application", appServiceManager)
						espXmlmc.SetParam("entity", "Requests")
						espXmlmc.OpenElement("primaryEntityData")
						espXmlmc.OpenElement("record")
						espXmlmc.SetParam("h_pk_reference", strNewCallRef)
						espXmlmc.SetParam("h_bpm_id", xmlRespon.Identifier)
						espXmlmc.CloseElement("record")
						espXmlmc.CloseElement("primaryEntityData")

						XMLBPMUpdate, xmlmcErr := espXmlmc.Invoke("data", "entityUpdateRecord")
						if xmlmcErr != nil {
							//log.Fatal(xmlmcErr)
							logger(4, "Unable to associated spawned BPM to request ["+strNewCallRef+"]: "+fmt.Sprintf("%v", xmlmcErr), false)
						}
						var xmlRespon xmlmcResponse

						errBPMSpawn := xml.Unmarshal([]byte(XMLBPMUpdate), &xmlRespon)
						if errBPMSpawn != nil {
							logger(4, "Unable to read response from Hornbill instance:"+fmt.Sprintf("%v", errBPMSpawn), false)
							return false, "No"
						}
						if xmlRespon.MethodResult != "ok" {
							logger(4, "Unable to associate BPM to Request: "+xmlRespon.State.ErrorRet, false)
						}
					}
				}
			}

			// Now handle calls in an On Hold status
			if strStatus == "status.onHold" {
				espXmlmc.SetParam("requestId", strNewCallRef)
				espXmlmc.SetParam("onHoldUntil", strClosedDate)
				espXmlmc.SetParam("strReason", "Request imported from Supportworks in an On Hold status. See Historical Request Updates for further information.")
				XMLBPM, xmlmcErr := espXmlmc.Invoke("apps/"+appServiceManager+"/Requests", "holdRequest")
				if xmlmcErr != nil {
					//log.Fatal(xmlmcErr)
					logger(4, "Unable to place request on hold ["+strNewCallRef+"] : "+fmt.Sprintf("%v", xmlmcErr), false)
				}
				var xmlRespon xmlmcResponse

				errLogDate := xml.Unmarshal([]byte(XMLBPM), &xmlRespon)
				if errLogDate != nil {
					logger(4, "Unable to place request on hold ["+strNewCallRef+"] : "+fmt.Sprintf("%v", errLogDate), false)
				}
				if xmlRespon.MethodResult != "ok" {
					logger(4, "Unable to place request on hold ["+strNewCallRef+"] : "+xmlRespon.State.ErrorRet, false)
				}
			}
		}
	} else {
		//-- DEBUG XML TO LOG FILE
		var XMLSTRING = espXmlmc.GetParam()
		logger(1, "Request Log XML "+fmt.Sprintf("%s", XMLSTRING), false)
		counters.Lock()
		counters.createdSkipped++
		counters.Unlock()
		espXmlmc.ClearParam()
		return true, "Dry Run"
	}

	//-- If request logged successfully :
	//Get the Call Diary Updates from Supportworks and build the Historical Updates against the SM request
	if boolCallLoggedOK == true && strNewCallRef != "" {
		applyHistoricalUpdates(strNewCallRef, swCallID)
	}

	return boolCallLoggedOK, strNewCallRef
}

//applyHistoricalUpdates - takes call diary records from Supportworks, imports to Hornbill as Historical Updates
func applyHistoricalUpdates(newCallRef, swCallRef string) bool {
	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return false
	}

	//Connect to the JSON specified DB
	db, err := sqlx.Open(appDBDriver, connStrAppDB)
	defer db.Close()
	if err != nil {
		logger(4, " [DATABASE] Database Connection Error for Historical Updates: "+fmt.Sprintf("%v", err), false)
		return false
	}
	//Check connection is open
	err = db.Ping()
	if err != nil {
		logger(4, " [DATABASE] [PING] Database Connection Error for Historical Updates: "+fmt.Sprintf("%v", err), false)
		return false
	}
	logger(3, "[DATABASE] Connection Successful", false)
	mutex.Lock()
	logger(3, "[DATABASE] Running query for Historical Updates of call "+swCallRef+". Please wait...", false)
	//build query
	sqlDiaryQuery := "SELECT updatetimex, repid, groupid, udsource, udcode, udtype, updatetxt, udindex, timespent "
	sqlDiaryQuery = sqlDiaryQuery + " FROM updatedb WHERE callref = " + swCallRef + " ORDER BY udindex DESC"
	logger(3, "[DATABASE} Diary Query: "+sqlDiaryQuery, false)
	mutex.Unlock()
	//Run Query
	rows, err := db.Queryx(sqlDiaryQuery)
	if err != nil {
		logger(4, " Database Query Error: "+fmt.Sprintf("%v", err), false)
		return false
	}
	//Process each call diary entry, insert in to Hornbill
	for rows.Next() {
		diaryEntry := make(map[string]interface{})
		err = rows.MapScan(diaryEntry)
		if err != nil {
			logger(4, "Unable to retrieve data from SQL query: "+fmt.Sprintf("%v", err), false)
		} else {
			//Update Time - EPOCH to Date/Time Conversion
			diaryTime := ""
			if diaryEntry["updatetimex"] != nil {
				diaryTimex := ""
				if updateTime, ok := diaryEntry["updatetimex"].(int64); ok {
					diaryTimex = strconv.FormatInt(updateTime, 10)
				} else {
					diaryTimex = fmt.Sprintf("%+s", diaryEntry["updatetimex"])
				}
				diaryTime = epochToDateTime(diaryTimex)
			}

			//Check for source/code/text having nil value
			diarySource := ""
			if diaryEntry["udsource"] != nil {
				diarySource = fmt.Sprintf("%+s", diaryEntry["udsource"])
			}

			diaryCode := ""
			if diaryEntry["udcode"] != nil {
				diaryCode = fmt.Sprintf("%+s", diaryEntry["udcode"])
			}

			diaryText := ""
			if diaryEntry["updatetxt"] != nil {
				diaryText = fmt.Sprintf("%+s", diaryEntry["updatetxt"])
				diaryText = html.EscapeString(diaryText)
			}

			diaryIndex := ""
			if diaryEntry["udindex"] != nil {
				if updateIndex, ok := diaryEntry["udindex"].(int64); ok {
					diaryIndex = strconv.FormatInt(updateIndex, 10)
				} else {
					diaryIndex = fmt.Sprintf("%+s", diaryEntry["udindex"])
				}
			}

			diaryTimeSpent := ""
			if diaryEntry["timespent"] != nil {
				if updateSpent, ok := diaryEntry["timespent"].(int64); ok {
					diaryTimeSpent = strconv.FormatInt(updateSpent, 10)
				} else {
					diaryTimeSpent = fmt.Sprintf("%+s", diaryEntry["timespent"])
				}
			}

			diaryType := ""
			if diaryEntry["udtype"] != nil {
				if updateType, ok := diaryEntry["udtype"].(int64); ok {
					diaryType = strconv.FormatInt(updateType, 10)
				} else {
					diaryType = fmt.Sprintf("%+s", diaryEntry["udtype"])
				}
			}

			espXmlmc.SetParam("application", appServiceManager)
			espXmlmc.SetParam("entity", "RequestHistoricUpdates")
			espXmlmc.OpenElement("primaryEntityData")
			espXmlmc.OpenElement("record")
			espXmlmc.SetParam("h_fk_reference", newCallRef)
			espXmlmc.SetParam("h_updatedate", diaryTime)
			if diaryTimeSpent != "" && diaryTimeSpent != "0" {
				espXmlmc.SetParam("h_timespent", diaryTimeSpent)
			}
			if diaryType != "" {
				espXmlmc.SetParam("h_updatetype", diaryType)
			}
			espXmlmc.SetParam("h_updatebytype", "1")
			espXmlmc.SetParam("h_updateindex", diaryIndex)
			espXmlmc.SetParam("h_updateby", fmt.Sprintf("%+s", diaryEntry["repid"]))
			espXmlmc.SetParam("h_updatebyname", fmt.Sprintf("%+s", diaryEntry["repid"]))
			espXmlmc.SetParam("h_updatebygroup", fmt.Sprintf("%+s", diaryEntry["groupid"]))
			if diaryCode != "" {
				espXmlmc.SetParam("h_actiontype", diaryCode)
			}
			if diarySource != "" {
				espXmlmc.SetParam("h_actionsource", diarySource)
			}
			if diaryText != "" {
				espXmlmc.SetParam("h_description", diaryText)
			}
			espXmlmc.CloseElement("record")
			espXmlmc.CloseElement("primaryEntityData")

			//-- Check for Dry Run
			if configDryRun != true {
				XMLUpdate, xmlmcErr := espXmlmc.Invoke("data", "entityAddRecord")
				if xmlmcErr != nil {
					//log.Fatal(xmlmcErr)
					logger(3, "Unable to add Historical Call Diary Update: "+fmt.Sprintf("%v", xmlmcErr), false)
				}
				var xmlRespon xmlmcResponse
				errXMLMC := xml.Unmarshal([]byte(XMLUpdate), &xmlRespon)
				if errXMLMC != nil {
					logger(4, "Unable to read response from Hornbill instance:"+fmt.Sprintf("%v", errXMLMC), false)
				}
				if xmlRespon.MethodResult != "ok" {
					logger(3, "Unable to add Historical Call Diary Update: "+xmlRespon.State.ErrorRet, false)
				}
			} else {
				//-- DEBUG XML TO LOG FILE
				var XMLSTRING = espXmlmc.GetParam()
				logger(1, "Request Historical Update XML "+fmt.Sprintf("%s", XMLSTRING), false)
				counters.Lock()
				counters.createdSkipped++
				counters.Unlock()
				espXmlmc.ClearParam()
				return true
			}
		}
	}
	defer rows.Close()
	return true
}

// getFieldValue --Retrieve field value from mapping via SQL record map
func getFieldValue(v string, u map[string]interface{}) string {
	fieldMap := v
	//-- Match $variable from String
	re1, err := regexp.Compile(`\[(.*?)\]`)
	if err != nil {
		color.Red("[ERROR] %v", err)
	}

	result := re1.FindAllString(fieldMap, 100)
	valFieldMap := ""
	//-- Loop Matches
	for _, val := range result {
		valFieldMap = ""
		valFieldMap = strings.Replace(val, "[", "", 1)
		valFieldMap = strings.Replace(valFieldMap, "]", "", 1)
		if valFieldMap == "oldCallRef" {
			valFieldMap = "h_formattedcallref"
			if u[valFieldMap] != nil {

				if valField, ok := u[valFieldMap].(int64); ok {
					valFieldMap = strconv.FormatInt(valField, 10)
				} else {
					valFieldMap = fmt.Sprintf("%+s", u[valFieldMap])
				}

				if valFieldMap != "<nil>" {
					fieldMap = strings.Replace(fieldMap, val, valFieldMap, 1)
				}

			} else {
				valFieldMap = "callref"
				if u[valFieldMap] != nil {

					if valField, ok := u[valFieldMap].(int64); ok {
						valFieldMap = strconv.FormatInt(valField, 10)
					} else {
						valFieldMap = fmt.Sprintf("%+s", u[valFieldMap])
					}

					if valFieldMap != "<nil>" {
						fieldMap = strings.Replace(fieldMap, val, padCallRef(valFieldMap, "F", 7), 1)
					}
				} else {
					fieldMap = strings.Replace(fieldMap, val, "", 1)
				}
			}
		} else {
			if u[valFieldMap] != nil {

				if valField, ok := u[valFieldMap].(int64); ok {
					valFieldMap = strconv.FormatInt(valField, 10)
				} else {
					valFieldMap = fmt.Sprintf("%+s", u[valFieldMap])
				}

				if valFieldMap != "<nil>" {
					fieldMap = strings.Replace(fieldMap, val, valFieldMap, 1)
				}
			} else {
				fieldMap = strings.Replace(fieldMap, val, "", 1)
			}
		}
	}
	return fieldMap
}

//getSiteID takes the Call Record and returns a correct Site ID if one exists on the Instance
func getSiteID(callMap map[string]interface{}) (string, string) {
	siteID := ""
	siteNameMapping := fmt.Sprintf("%v", mapGenericConf.CoreFieldMapping["h_site_id"])
	siteName := getFieldValue(siteNameMapping, callMap)
	if siteName != "" {
		siteIsInCache, SiteIDCache := recordInCache(siteName, "Site")
		//-- Check if we have cached the site already
		if siteIsInCache {
			siteID = SiteIDCache
		} else {
			siteIsOnInstance, SiteIDInstance := searchSite(siteName)
			//-- If Returned set output
			if siteIsOnInstance {
				siteID = strconv.Itoa(SiteIDInstance)
			}
		}
	}
	return siteID, siteName
}

//getCallServiceID takes the Call Record and returns a correct Service ID if one exists on the Instance
func getCallServiceID(swService string) string {
	serviceID := ""
	serviceName := ""
	if swImportConf.ServiceMapping[swService] != nil {
		serviceName = fmt.Sprintf("%s", swImportConf.ServiceMapping[swService])

		if serviceName != "" {
			serviceID = getServiceID(serviceName)
		}
	}
	return serviceID
}

//getServiceID takes a Service Name string and returns a correct Service ID if one exists in the cache or on the Instance
func getServiceID(serviceName string) string {
	serviceID := ""
	if serviceName != "" {
		serviceIsInCache, ServiceIDCache := recordInCache(serviceName, "Service")
		//-- Check if we have cached the Service already
		if serviceIsInCache {
			serviceID = ServiceIDCache
		} else {
			serviceIsOnInstance, ServiceIDInstance := searchService(serviceName)
			//-- If Returned set output
			if serviceIsOnInstance {
				serviceID = strconv.Itoa(ServiceIDInstance)
			}
		}
	}
	return serviceID
}

//getCallPriorityID takes the Call Record and returns a correct Priority ID if one exists on the Instance
func getCallPriorityID(strPriorityName string) (string, string) {
	priorityID := ""
	if swImportConf.PriorityMapping[strPriorityName] != nil {
		strPriorityName = fmt.Sprintf("%s", swImportConf.PriorityMapping[strPriorityName])
		if strPriorityName != "" {
			priorityID = getPriorityID(strPriorityName)
		}
	}
	return priorityID, strPriorityName
}

//getPriorityID takes a Priority Name string and returns a correct Priority ID if one exists in the cache or on the Instance
func getPriorityID(priorityName string) string {
	priorityID := ""
	if priorityName != "" {
		priorityIsInCache, PriorityIDCache := recordInCache(priorityName, "Priority")
		//-- Check if we have cached the Priority already
		if priorityIsInCache {
			priorityID = PriorityIDCache
		} else {
			priorityIsOnInstance, PriorityIDInstance := searchPriority(priorityName)
			//-- If Returned set output
			if priorityIsOnInstance {
				priorityID = strconv.Itoa(PriorityIDInstance)
			}
		}
	}
	return priorityID
}

//getCallTeamID takes the Call Record and returns a correct Team ID if one exists on the Instance
func getCallTeamID(swTeamID string) (string, string) {
	teamID := ""
	teamName := ""
	if swImportConf.TeamMapping[swTeamID] != nil {
		teamName = fmt.Sprintf("%s", swImportConf.TeamMapping[swTeamID])
		if teamName != "" {
			teamID = getTeamID(teamName)
		}
	}
	return teamID, teamName
}

//getTeamID takes a Team Name string and returns a correct Team ID if one exists in the cache or on the Instance
func getTeamID(teamName string) string {
	teamID := ""
	if teamName != "" {
		teamIsInCache, TeamIDCache := recordInCache(teamName, "Team")
		//-- Check if we have cached the Team already
		if teamIsInCache {
			teamID = TeamIDCache
		} else {
			teamIsOnInstance, TeamIDInstance := searchTeam(teamName)
			//-- If Returned set output
			if teamIsOnInstance {
				teamID = TeamIDInstance
			}
		}
	}
	return teamID
}

//getCallCategoryID takes the Call Record and returns a correct Category ID if one exists on the Instance
func getCallCategoryID(callMap map[string]interface{}, categoryGroup string) (string, string) {
	categoryID := ""
	categoryString := ""
	categoryNameMapping := ""
	categoryCode := ""
	if categoryGroup == "Request" {
		categoryNameMapping = fmt.Sprintf("%v", mapGenericConf.CoreFieldMapping["h_category_id"])
		categoryCode = getFieldValue(categoryNameMapping, callMap)
		if swImportConf.CategoryMapping[categoryCode] != nil {
			//Get Category Code from JSON mapping
			categoryCode = fmt.Sprintf("%s", swImportConf.CategoryMapping[categoryCode])
		} else {
			//Mapping doesn't exist - replace hyphens from SW Profile code with another string, and try to use this
			//SMProfileCodeSeperator allows us to specify in the config, the seperator used within Service Manager
			//profile codes
			categoryCode = strings.Replace(categoryCode, "-", swImportConf.SMProfileCodeSeperator, -1)
		}

	} else {
		categoryNameMapping = fmt.Sprintf("%v", mapGenericConf.CoreFieldMapping["h_closure_category_id"])
		categoryCode = getFieldValue(categoryNameMapping, callMap)
		if swImportConf.ResolutionCategoryMapping[categoryCode] != nil {
			//Get Category Code from JSON mapping
			categoryCode = fmt.Sprintf("%s", swImportConf.ResolutionCategoryMapping[categoryCode])
		} else {
			//Mapping doesn't exist - replace hyphens from SW Profile code with colon, and try to use this
			categoryCode = strings.Replace(categoryCode, "-", swImportConf.SMProfileCodeSeperator, -1)
		}
	}
	if categoryCode != "" {
		categoryID, categoryString = getCategoryID(categoryCode, categoryGroup)
	}
	return categoryID, categoryString
}

//getCategoryID takes a Category Code string and returns a correct Category ID if one exists in the cache or on the Instance
func getCategoryID(categoryCode, categoryGroup string) (string, string) {

	categoryID := ""
	categoryString := ""
	if categoryCode != "" {
		categoryIsInCache, CategoryIDCache, CategoryNameCache := categoryInCache(categoryCode, categoryGroup+"Category")
		//-- Check if we have cached the Category already
		if categoryIsInCache {
			categoryID = CategoryIDCache
			categoryString = CategoryNameCache
		} else {
			categoryIsOnInstance, CategoryIDInstance, CategoryStringInstance := searchCategory(categoryCode, categoryGroup)
			//-- If Returned set output
			if categoryIsOnInstance {
				categoryID = CategoryIDInstance
				categoryString = CategoryStringInstance
			} else {
				logger(4, "[CATEGORY] "+categoryGroup+" Category ["+categoryCode+"] is not on instance.", false)
			}
		}
	}
	return categoryID, categoryString
}

//doesAnalystExist takes an Analyst ID string and returns a true if one exists in the cache or on the Instance
func doesAnalystExist(analystID string) bool {
	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return false
	}
	boolAnalystExists := false
	if analystID != "" {
		analystIsInCache, strReturn := recordInCache(analystID, "Analyst")
		//-- Check if we have cached the Analyst already
		if analystIsInCache && strReturn != "" {
			boolAnalystExists = true
		} else {
			//Get Analyst Info
			espXmlmc.SetParam("userId", analystID)

			XMLAnalystSearch, xmlmcErr := espXmlmc.Invoke("admin", "userGetInfo")
			if xmlmcErr != nil {
				logger(4, "Unable to Search for Request Owner ["+analystID+"]: "+fmt.Sprintf("%v", xmlmcErr), true)
			}

			var xmlRespon xmlmcAnalystListResponse
			err := xml.Unmarshal([]byte(XMLAnalystSearch), &xmlRespon)
			if err != nil {
				logger(4, "Unable to Search for Request Owner ["+analystID+"]: "+fmt.Sprintf("%v", err), false)
			} else {
				if xmlRespon.MethodResult != "ok" {
					//Analyst most likely does not exist
					logger(4, "Unable to Search for Request Owner ["+analystID+"]: "+xmlRespon.State.ErrorRet, false)
				} else {
					//-- Check Response
					if xmlRespon.AnalystFullName != "" {
						boolAnalystExists = true
						//-- Add Analyst to Cache
						var newAnalystForCache analystListStruct
						newAnalystForCache.AnalystID = analystID
						newAnalystForCache.AnalystName = xmlRespon.AnalystFullName
						analystNamedMap := []analystListStruct{newAnalystForCache}
						mutexAnalysts.Lock()
						analysts = append(analysts, analystNamedMap...)
						mutexAnalysts.Unlock()
					}
				}
			}
		}
	}
	return boolAnalystExists
}

//doesCustomerExist takes a Customer ID string and returns a true if one exists in the cache or on the Instance
func doesCustomerExist(customerID string) bool {
	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return false
	}
	boolCustomerExists := false
	if customerID != "" {
		customerIsInCache, strReturn := recordInCache(customerID, "Customer")
		//-- Check if we have cached the Analyst already
		if customerIsInCache && strReturn != "" {
			boolCustomerExists = true
		} else {
			//Get Analyst Info
			espXmlmc.SetParam("customerId", customerID)
			espXmlmc.SetParam("customerType", swImportConf.CustomerType)
			XMLCustomerSearch, xmlmcErr := espXmlmc.Invoke("apps/"+appServiceManager, "shrGetCustomerDetails")
			if xmlmcErr != nil {
				logger(4, "Unable to Search for Customer ["+customerID+"]: "+fmt.Sprintf("%v", xmlmcErr), true)
			}

			var xmlRespon xmlmcCustomerListResponse
			err := xml.Unmarshal([]byte(XMLCustomerSearch), &xmlRespon)
			if err != nil {
				logger(4, "Unable to Search for Customer ["+customerID+"]: "+fmt.Sprintf("%v", err), false)
			} else {
				if xmlRespon.MethodResult != "ok" {
					//Customer most likely does not exist
					logger(4, "Unable to Search for Customer ["+customerID+"]: "+xmlRespon.State.ErrorRet, false)
				} else {
					//-- Check Response
					if xmlRespon.CustomerFirstName != "" {
						boolCustomerExists = true
						//-- Add Customer to Cache
						var newCustomerForCache customerListStruct
						newCustomerForCache.CustomerID = customerID
						newCustomerForCache.CustomerName = xmlRespon.CustomerFirstName + " " + xmlRespon.CustomerLastName
						customerNamedMap := []customerListStruct{newCustomerForCache}
						mutexCustomers.Lock()
						customers = append(customers, customerNamedMap...)
						mutexCustomers.Unlock()
					}
				}
			}
		}
	}
	return boolCustomerExists
}

// recordInCache -- Function to check if passed-thorugh record name has been cached
// if so, pass back the Record ID
func recordInCache(recordName, recordType string) (bool, string) {
	boolReturn := false
	strReturn := ""
	switch recordType {
	case "Service":
		//-- Check if record in Service Cache
		mutexServices.Lock()
		for _, service := range services {
			if service.ServiceName == recordName {
				boolReturn = true
				strReturn = strconv.Itoa(service.ServiceID)
			}
		}
		mutexServices.Unlock()
	case "Priority":
		//-- Check if record in Priority Cache
		mutexPriorities.Lock()
		for _, priority := range priorities {
			if priority.PriorityName == recordName {
				boolReturn = true
				strReturn = strconv.Itoa(priority.PriorityID)
			}
		}
		mutexPriorities.Unlock()
	case "Site":
		//-- Check if record in Site Cache
		mutexSites.Lock()
		for _, site := range sites {
			if site.SiteName == recordName {
				boolReturn = true
				strReturn = strconv.Itoa(site.SiteID)
			}
		}
		mutexSites.Unlock()
	case "Team":
		//-- Check if record in Team Cache
		mutexTeams.Lock()
		for _, team := range teams {
			if team.TeamName == recordName {
				boolReturn = true
				strReturn = team.TeamID
			}
		}
		mutexTeams.Unlock()
	case "Analyst":
		//-- Check if record in Analyst Cache
		mutexAnalysts.Lock()
		for _, analyst := range analysts {
			if analyst.AnalystID == recordName {
				boolReturn = true
				strReturn = analyst.AnalystName
			}
		}
		mutexAnalysts.Unlock()
	case "Customer":
		//-- Check if record in Customer Cache
		mutexCustomers.Lock()
		for _, customer := range customers {
			if customer.CustomerID == recordName {
				boolReturn = true
				strReturn = customer.CustomerName
			}
		}
		mutexCustomers.Unlock()
	}
	return boolReturn, strReturn
}

// categoryInCache -- Function to check if passed-thorugh category been cached
// if so, pass back the Category ID and Full Name
func categoryInCache(recordName, recordType string) (bool, string, string) {
	boolReturn := false
	idReturn := ""
	strReturn := ""
	switch recordType {
	case "RequestCategory":
		//-- Check if record in Category Cache
		mutexCategories.Lock()
		for _, category := range categories {
			if category.CategoryCode == recordName {
				boolReturn = true
				idReturn = category.CategoryID
				strReturn = category.CategoryName
			}
		}
		mutexCategories.Unlock()
	case "ClosureCategory":
		//-- Check if record in Category Cache
		mutexCloseCategories.Lock()
		for _, category := range closeCategories {
			if category.CategoryCode == recordName {
				boolReturn = true
				idReturn = category.CategoryID
				strReturn = category.CategoryName
			}
		}
		mutexCloseCategories.Unlock()
	}
	return boolReturn, idReturn, strReturn
}

// seachSite -- Function to check if passed-through  site  name is on the instance
func searchSite(siteName string) (bool, int) {
	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return false, 0
	}

	boolReturn := false
	intReturn := 0
	//-- ESP Query for site
	espXmlmc.SetParam("entity", "Site")
	espXmlmc.SetParam("matchScope", "all")
	espXmlmc.OpenElement("searchFilter")
	espXmlmc.SetParam("h_site_name", siteName)
	espXmlmc.CloseElement("searchFilter")
	espXmlmc.SetParam("maxResults", "1")

	XMLSiteSearch, xmlmcErr := espXmlmc.Invoke("data", "entityBrowseRecords")
	if xmlmcErr != nil {
		logger(4, "Unable to Search for Site: "+fmt.Sprintf("%v", xmlmcErr), false)
		return boolReturn, intReturn
		//log.Fatal(xmlmcErr)
	}
	var xmlRespon xmlmcSiteListResponse

	err = xml.Unmarshal([]byte(XMLSiteSearch), &xmlRespon)
	if err != nil {
		logger(4, "Unable to Search for Site: "+fmt.Sprintf("%v", err), false)
	} else {
		if xmlRespon.MethodResult != "ok" {
			logger(4, "Unable to Search for Site: "+xmlRespon.State.ErrorRet, false)
		} else {
			//-- Check Response
			if xmlRespon.SiteName != "" {
				if strings.ToLower(xmlRespon.SiteName) == strings.ToLower(siteName) {
					intReturn = xmlRespon.SiteID
					boolReturn = true
					//-- Add Site to Cache
					var newSiteForCache siteListStruct
					newSiteForCache.SiteID = intReturn
					newSiteForCache.SiteName = siteName
					siteNamedMap := []siteListStruct{newSiteForCache}
					mutexSites.Lock()
					sites = append(sites, siteNamedMap...)
					mutexSites.Unlock()
				}
			}
		}
	}
	return boolReturn, intReturn
}

// seachPriority -- Function to check if passed-through priority name is on the instance
func searchPriority(priorityName string) (bool, int) {
	boolReturn := false
	intReturn := 0
	//-- ESP Query for Priority
	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return false, 0
	}

	espXmlmc.SetParam("application", appServiceManager)
	espXmlmc.SetParam("entity", "Priority")
	espXmlmc.SetParam("matchScope", "all")
	espXmlmc.OpenElement("searchFilter")
	espXmlmc.SetParam("h_priorityname", priorityName)
	espXmlmc.CloseElement("searchFilter")
	espXmlmc.SetParam("maxResults", "1")

	XMLPrioritySearch, xmlmcErr := espXmlmc.Invoke("data", "entityBrowseRecords")
	if xmlmcErr != nil {
		logger(4, "Unable to Search for Priority: "+fmt.Sprintf("%v", xmlmcErr), false)
		return boolReturn, intReturn
		//log.Fatal(xmlmcErr)
	}
	var xmlRespon xmlmcPriorityListResponse

	err = xml.Unmarshal([]byte(XMLPrioritySearch), &xmlRespon)
	if err != nil {
		logger(4, "Unable to Search for Priority: "+fmt.Sprintf("%v", err), false)
	} else {
		if xmlRespon.MethodResult != "ok" {
			logger(4, "Unable to Search for Priority: "+xmlRespon.State.ErrorRet, false)
		} else {
			//-- Check Response
			if xmlRespon.PriorityName != "" {
				if strings.ToLower(xmlRespon.PriorityName) == strings.ToLower(priorityName) {
					intReturn = xmlRespon.PriorityID
					boolReturn = true
					//-- Add Priority to Cache
					var newPriorityForCache priorityListStruct
					newPriorityForCache.PriorityID = intReturn
					newPriorityForCache.PriorityName = priorityName
					priorityNamedMap := []priorityListStruct{newPriorityForCache}
					mutexPriorities.Lock()
					priorities = append(priorities, priorityNamedMap...)
					mutexPriorities.Unlock()
				}
			}
		}
	}
	return boolReturn, intReturn
}

// seachService -- Function to check if passed-through service name is on the instance
func searchService(serviceName string) (bool, int) {
	boolReturn := false
	intReturn := 0
	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return false, 0
	}

	//-- ESP Query for service
	espXmlmc.SetParam("application", appServiceManager)
	espXmlmc.SetParam("entity", "Services")
	espXmlmc.SetParam("matchScope", "all")
	espXmlmc.OpenElement("searchFilter")
	espXmlmc.SetParam("h_servicename", serviceName)
	espXmlmc.CloseElement("searchFilter")
	espXmlmc.SetParam("maxResults", "1")

	XMLServiceSearch, xmlmcErr := espXmlmc.Invoke("data", "entityBrowseRecords")
	if xmlmcErr != nil {
		logger(4, "Unable to Search for Service: "+fmt.Sprintf("%v", xmlmcErr), false)
		//log.Fatal(xmlmcErr)
		return boolReturn, intReturn
	}
	var xmlRespon xmlmcServiceListResponse

	err = xml.Unmarshal([]byte(XMLServiceSearch), &xmlRespon)
	if err != nil {
		logger(4, "Unable to Search for Service: "+fmt.Sprintf("%v", err), false)
	} else {
		if xmlRespon.MethodResult != "ok" {
			logger(4, "Unable to Search for Service: "+xmlRespon.State.ErrorRet, false)
		} else {
			//-- Check Response
			if xmlRespon.ServiceName != "" {
				if strings.ToLower(xmlRespon.ServiceName) == strings.ToLower(serviceName) {
					intReturn = xmlRespon.ServiceID
					boolReturn = true
					//-- Add Service to Cache
					var newServiceForCache serviceListStruct
					newServiceForCache.ServiceID = intReturn
					newServiceForCache.ServiceName = serviceName
					newServiceForCache.ServiceBPMIncident = xmlRespon.BPMIncident
					newServiceForCache.ServiceBPMService = xmlRespon.BPMService
					newServiceForCache.ServiceBPMChange = xmlRespon.BPMChange
					newServiceForCache.ServiceBPMProblem = xmlRespon.BPMProblem
					newServiceForCache.ServiceBPMKnownError = xmlRespon.BPMKnownError
					serviceNamedMap := []serviceListStruct{newServiceForCache}
					mutexServices.Lock()
					services = append(services, serviceNamedMap...)
					mutexServices.Unlock()
				}
			}
		}
	}
	//Return Service ID once cached - we can now use this in the calling function to get all details from cache
	return boolReturn, intReturn
}

// searchTeam -- Function to check if passed-through support team name is on the instance
func searchTeam(teamName string) (bool, string) {
	boolReturn := false
	strReturn := ""
	//-- ESP Query for team
	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return false, "Unable to create connection"
	}

	espXmlmc.SetParam("entity", "Groups")
	espXmlmc.SetParam("matchScope", "all")
	espXmlmc.OpenElement("searchFilter")
	espXmlmc.SetParam("h_name", teamName)
	espXmlmc.CloseElement("searchFilter")
	espXmlmc.SetParam("maxResults", "1")

	XMLTeamSearch, xmlmcErr := espXmlmc.Invoke("data", "entityBrowseRecords")
	if xmlmcErr != nil {
		logger(4, "Unable to Search for Team: "+fmt.Sprintf("%v", xmlmcErr), true)
		//log.Fatal(xmlmcErr)
		return boolReturn, strReturn
	}
	var xmlRespon xmlmcTeamListResponse

	err = xml.Unmarshal([]byte(XMLTeamSearch), &xmlRespon)
	if err != nil {
		logger(4, "Unable to Search for Team: "+fmt.Sprintf("%v", err), true)
	} else {
		if xmlRespon.MethodResult != "ok" {
			logger(4, "Unable to Search for Team: "+xmlRespon.State.ErrorRet, true)
		} else {
			//-- Check Response
			if xmlRespon.TeamName != "" {
				if strings.ToLower(xmlRespon.TeamName) == strings.ToLower(teamName) {
					strReturn = xmlRespon.TeamID
					boolReturn = true
					//-- Add Team to Cache
					var newTeamForCache teamListStruct
					newTeamForCache.TeamID = strReturn
					newTeamForCache.TeamName = teamName
					teamNamedMap := []teamListStruct{newTeamForCache}
					mutexTeams.Lock()
					teams = append(teams, teamNamedMap...)
					mutexTeams.Unlock()
				}
			}
		}
	}
	return boolReturn, strReturn
}

// seachCategory -- Function to check if passed-through support category name is on the instance
func searchCategory(categoryCode, categoryGroup string) (bool, string, string) {
	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return false, "Unable to create connection", ""
	}

	boolReturn := false
	idReturn := ""
	strReturn := ""
	//-- ESP Query for category
	espXmlmc.SetParam("codeGroup", categoryGroup)
	espXmlmc.SetParam("code", categoryCode)
	var XMLSTRING = espXmlmc.GetParam()
	XMLCategorySearch, xmlmcErr := espXmlmc.Invoke("data", "profileCodeLookup")
	if xmlmcErr != nil {
		logger(4, "XMLMC API Invoke Failed for "+categoryGroup+" Category ["+categoryCode+"]: "+fmt.Sprintf("%v", xmlmcErr), false)
		logger(1, "Category Search XML "+fmt.Sprintf("%s", XMLSTRING), false)
		return boolReturn, idReturn, strReturn
	}
	var xmlRespon xmlmcCategoryListResponse

	err = xml.Unmarshal([]byte(XMLCategorySearch), &xmlRespon)
	if err != nil {
		logger(4, "Unable to unmarshal response for "+categoryGroup+" Category: "+fmt.Sprintf("%v", err), false)
		logger(1, "Category Search XML "+fmt.Sprintf("%s", XMLSTRING), false)
	} else {
		if xmlRespon.MethodResult != "ok" {
			logger(4, "Unable to Search for "+categoryGroup+" Category ["+categoryCode+"]: ["+fmt.Sprintf("%v", xmlRespon.MethodResult)+"] "+xmlRespon.State.ErrorRet, false)
			logger(1, "Category Search XML "+fmt.Sprintf("%s", XMLSTRING), false)
		} else {
			//-- Check Response
			if xmlRespon.CategoryName != "" {
				strReturn = xmlRespon.CategoryName
				idReturn = xmlRespon.CategoryID
				logger(3, "[CATEGORY] [SUCCESS] Methodcall result OK for "+categoryGroup+" Category ["+categoryCode+"] : ["+strReturn+"]", false)
				boolReturn = true
				//-- Add Category to Cache
				var newCategoryForCache categoryListStruct
				newCategoryForCache.CategoryID = idReturn
				newCategoryForCache.CategoryCode = categoryCode
				newCategoryForCache.CategoryName = strReturn
				categoryNamedMap := []categoryListStruct{newCategoryForCache}
				switch categoryGroup {
				case "Request":
					mutexCategories.Lock()
					categories = append(categories, categoryNamedMap...)
					mutexCategories.Unlock()
				case "Closure":
					mutexCloseCategories.Lock()
					closeCategories = append(closeCategories, categoryNamedMap...)
					mutexCloseCategories.Unlock()
				}
			} else {
				logger(3, "[CATEGORY] [FAIL] Methodcall result OK for "+categoryGroup+" Category ["+categoryCode+"] but category name blank: ["+xmlRespon.CategoryID+"] ["+xmlRespon.CategoryName+"]", false)
				logger(3, "[CATEGORY] [FAIL] Category Search XML "+fmt.Sprintf("%s", XMLSTRING), false)
			}
		}
	}
	return boolReturn, idReturn, strReturn
}

//padCalLRef -- Function to pad Call Reference to specified digits, adding an optional prefix
func padCallRef(strIntCallRef, prefix string, length int) (paddedRef string) {
	if len(strIntCallRef) < length {
		padCount := length - len(strIntCallRef)
		strIntCallRef = strings.Repeat("0", padCount) + strIntCallRef
	}
	paddedRef = prefix + strIntCallRef
	return
}

//loadConfig -- Function to Load Configruation File
func loadConfig() (swImportConfStruct, bool) {
	boolLoadConf := true
	//-- Check Config File File Exists
	cwd, _ := os.Getwd()
	configurationFilePath := cwd + "/" + configFileName
	logger(1, "Loading Config File: "+configurationFilePath, false)
	if _, fileCheckErr := os.Stat(configurationFilePath); os.IsNotExist(fileCheckErr) {
		logger(4, "No Configuration File", true)
		os.Exit(102)
	}
	//-- Load Config File
	file, fileError := os.Open(configurationFilePath)
	//-- Check For Error Reading File
	if fileError != nil {
		logger(4, "Error Opening Configuration File: "+fmt.Sprintf("%v", fileError), true)
		boolLoadConf = false
	}

	//-- New Decoder
	decoder := json.NewDecoder(file)
	//-- New Var based on swImportConfStruct
	edbConf := swImportConfStruct{}
	//-- Decode JSON
	err := decoder.Decode(&edbConf)
	//-- Error Checking
	if err != nil {
		logger(4, "Error Decoding Configuration File: "+fmt.Sprintf("%v", err), true)
		boolLoadConf = false
	}
	//-- Return New Config
	return edbConf, boolLoadConf
}

//login -- XMLMC Login
//-- start ESP user session
func login() bool {
	logger(1, "Logging Into: "+swImportConf.HBConf.URL, false)
	logger(1, "UserName: "+swImportConf.HBConf.UserName, false)
	espXmlmc = apiLib.NewXmlmcInstance(swImportConf.HBConf.URL)

	espXmlmc.SetParam("userId", swImportConf.HBConf.UserName)
	espXmlmc.SetParam("password", base64.StdEncoding.EncodeToString([]byte(swImportConf.HBConf.Password)))
	XMLLogin, xmlmcErr := espXmlmc.Invoke("session", "userLogon")
	if xmlmcErr != nil {
		log.Fatal(xmlmcErr)
	}

	var xmlRespon xmlmcResponse
	err := xml.Unmarshal([]byte(XMLLogin), &xmlRespon)
	if err != nil {
		logger(4, "Unable to Login: "+fmt.Sprintf("%v", err), true)
		return false
	}
	if xmlRespon.MethodResult != "ok" {
		logger(4, "Unable to Login: "+xmlRespon.State.ErrorRet, true)
		return false
	}
	espLogger("---- Supportworks Call Import Utility V"+fmt.Sprintf("%v", version)+" ----", "debug")
	espLogger("Logged In As: "+swImportConf.HBConf.UserName, "debug")
	return true
}

//logout -- XMLMC Logout
//-- Adds details to log file, ends user ESP session
func logout() {
	//-- End output
	espLogger("Requests Logged: "+fmt.Sprintf("%d", counters.created), "debug")
	espLogger("Requests Skipped: "+fmt.Sprintf("%d", counters.createdSkipped), "debug")
	espLogger("Time Taken: "+fmt.Sprintf("%v", endTime), "debug")
	espLogger("---- Supportworks Call Import Complete ---- ", "debug")
	logger(1, "Logout", true)
	espXmlmc.Invoke("session", "userLogoff")
}

//buildConnectionString -- Build the connection string for the SQL driver
func buildConnectionString(strDataSource string) string {
	connectString := ""
	if strDataSource == "app" {
		//Build
		if appDBDriver == "" || swImportConf.SWAppDBConf.Server == "" || swImportConf.SWAppDBConf.Database == "" || swImportConf.SWAppDBConf.UserName == "" || swImportConf.SWAppDBConf.Port == 0 {
			logger(4, "Application Database configuration not set.", true)
			return ""
		}
		switch appDBDriver {
		case "mssql":
			connectString = "server=" + swImportConf.SWAppDBConf.Server
			connectString = connectString + ";database=" + swImportConf.SWAppDBConf.Database
			connectString = connectString + ";user id=" + swImportConf.SWAppDBConf.UserName
			connectString = connectString + ";password=" + swImportConf.SWAppDBConf.Password
			if swImportConf.SWAppDBConf.Encrypt == false {
				connectString = connectString + ";encrypt=disable"
			}
			if swImportConf.SWAppDBConf.Port != 0 {
				var dbPortSetting string
				dbPortSetting = strconv.Itoa(swImportConf.SWAppDBConf.Port)
				connectString = connectString + ";port=" + dbPortSetting
			}
		case "mysql":
			connectString = swImportConf.SWAppDBConf.UserName + ":" + swImportConf.SWAppDBConf.Password
			connectString = connectString + "@tcp(" + swImportConf.SWAppDBConf.Server + ":"
			if swImportConf.SWAppDBConf.Port != 0 {
				var dbPortSetting string
				dbPortSetting = strconv.Itoa(swImportConf.SWAppDBConf.Port)
				connectString = connectString + dbPortSetting
			} else {
				connectString = connectString + "3306"
			}
			connectString = connectString + ")/" + swImportConf.SWAppDBConf.Database

		case "mysql320":
			var dbPortSetting string
			dbPortSetting = strconv.Itoa(swImportConf.SWAppDBConf.Port)
			connectString = "tcp:" + swImportConf.SWAppDBConf.Server + ":" + dbPortSetting
			connectString = connectString + "*" + swImportConf.SWAppDBConf.Database + "/" + swImportConf.SWAppDBConf.UserName + "/" + swImportConf.SWAppDBConf.Password
		}
	} else if strDataSource == "cache" {
		//Build & return sw_systemdb connection string
		if swImportConf.SWServerAddress == "" {
			logger(4, "Supportworks Server Address not set.", true)
			return ""
		}
		if swImportConf.SWSystemDBConf.UserName == "" {
			logger(4, "System Database configuration not set.", true)
			return ""
		}
		connectString = "tcp:" + swImportConf.SWServerAddress + ":5002"
		connectString = connectString + "*sw_systemdb/" + swImportConf.SWSystemDBConf.UserName + "/" + swImportConf.SWSystemDBConf.Password

	}
	return connectString
}

// logger -- function to append to the current log file
func logger(t int, s string, outputtoCLI bool) {
	cwd, _ := os.Getwd()
	logPath := cwd + "/log"
	logFileName := logPath + "/SW_Call_Import_" + timeNow + ".log"

	//-- If Folder Does Not Exist then create it
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		err := os.Mkdir(logPath, 0777)
		if err != nil {
			color.Red("Error Creating Log Folder %q: %s \r", logPath, err)
			os.Exit(101)
		}
	}

	//-- Open Log File
	f, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0777)
	// don't forget to close it
	defer f.Close()
	if err != nil {
		color.Red("Error Creating Log File %q: %s \n", logFileName, err)
		os.Exit(100)
	}
	// assign it to the standard logger
	log.SetOutput(f)
	var errorLogPrefix string
	//-- Create Log Entry
	switch t {
	case 1:
		errorLogPrefix = "[DEBUG] "
		if outputtoCLI {
			color.Set(color.FgGreen)
			defer color.Unset()
		}
	case 2:
		errorLogPrefix = "[MESSAGE] "
		if outputtoCLI {
			color.Set(color.FgGreen)
			defer color.Unset()
		}
	case 3:
		if outputtoCLI {
			color.Set(color.FgGreen)
			defer color.Unset()
		}
	case 4:
		errorLogPrefix = "[ERROR] "
		if outputtoCLI {
			color.Set(color.FgRed)
			defer color.Unset()
		}
	case 5:
		errorLogPrefix = "[WARNING]"
		if outputtoCLI {
			color.Set(color.FgYellow)
			defer color.Unset()
		}
	case 6:
		if outputtoCLI {
			color.Set(color.FgYellow)
			defer color.Unset()
		}
	}

	if outputtoCLI {
		fmt.Printf("%v \n", errorLogPrefix+s)
	}

	log.Println(errorLogPrefix + s)
}

// espLogger -- Log to ESP
func espLogger(message string, severity string) {

	espXmlmc.SetParam("fileName", "SW_Call_Import")
	espXmlmc.SetParam("group", "general")
	espXmlmc.SetParam("severity", severity)
	espXmlmc.SetParam("message", message)
	espXmlmc.Invoke("system", "logMessage")
}

// SetInstance sets the Zone and Instance config from the passed-through strZone and instanceID values
func SetInstance(strZone string, instanceID string) {
	//-- Set Zone
	SetZone(strZone)
	//-- Set Instance
	xmlmcInstanceConfig.instance = instanceID
	return
}

// SetZone - sets the Instance Zone to Overide current live zone
func SetZone(zone string) {
	xmlmcInstanceConfig.zone = zone
	return
}

// getInstanceURL -- Function to build XMLMC End Point
func getInstanceURL() string {
	xmlmcInstanceConfig.url = "https://"
	xmlmcInstanceConfig.url += xmlmcInstanceConfig.zone
	xmlmcInstanceConfig.url += "api.hornbill.com/"
	xmlmcInstanceConfig.url += xmlmcInstanceConfig.instance
	xmlmcInstanceConfig.url += "/xmlmc/"
	return xmlmcInstanceConfig.url
}

//epochToDateTime - converts an EPOCH value STRING var in to a date-time format compatible with Hornbill APIs
func epochToDateTime(epochDateString string) string {
	dateTime := ""
	i, err := strconv.ParseInt(epochDateString, 10, 64)
	if err != nil {
		logger(5, "EPOCH String to Int conversion FAILED: "+fmt.Sprintf("%v", err), false)
	} else {
		dateTimeStr := fmt.Sprintf("%s", time.Unix(i, 0))
		for i := 0; i < 19; i++ {
			dateTime = dateTime + string(dateTimeStr[i])
		}
	}
	return dateTime
}

//NewEspXmlmcSession - New Xmlmc Session variable (Cloned Session)
func NewEspXmlmcSession() (*apiLib.XmlmcInstStruct, error) {
	time.Sleep(150 * time.Millisecond)
	espXmlmcLocal := apiLib.NewXmlmcInstance(swImportConf.HBConf.URL)
	espXmlmcLocal.SetSessionID(espXmlmc.GetSessionID())
	return espXmlmcLocal, nil
}
