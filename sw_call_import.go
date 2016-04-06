package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	_ "github.com/hornbill/go-mssqldb" //Microsoft SQL Server driver - v2005+
	"github.com/hornbill/goApiLib"
	_ "github.com/hornbill/mysql"    //MySQL v4.1 to v5.x and MariaDB driver
	_ "github.com/hornbill/mysql320" //MySQL v3.2.0 to v5 driver - Provides SWSQL (MySQL 4.0.16) support
	"github.com/hornbill/pb"
	"github.com/hornbill/sqlx"
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
	version           = 1.1
	appServiceManager = "com.hornbill.servicemanager"
	//Disk Space Declarations
	sizeKB        float64 = 1 << (10 * 1)
	sizeMB        float64 = 1 << (10 * 2)
	sizeGB        float64 = 1 << (10 * 3)
	sizeTB        float64 = 1 << (10 * 4)
	sizePB        float64 = 1 << (10 * 5)
	maxGoroutines int     = 5
)

var (
	appDBDriver         string
	cacheDBDriver       string
	arrCallsLogged      = make(map[string]string)
	arrClosedCalls      = make(map[string]string)
	arrCallDetailsMaps  = make([]map[string]interface{}, 0)
	arrSWStatus         = make(map[string]string)
	boolConfLoaded      bool
	boolProcessClass    bool
	configFileName      string
	configZone          string
	configDryRun        bool
	connStrSysDB        string
	connStrAppDB        string
	counters            counterTypeStruct
	mapGenericConf      swCallConfStruct
	currentCallRef      string
	analysts            []analystListStruct
	categories          []categoryListStruct
	closeCategories     []categoryListStruct
	priorities          []priorityListStruct
	services            []serviceListStruct
	sites               []siteListStruct
	teams               []teamListStruct
	strNewCallRef       string
	sqlCallQuery        string
	swImportConf        swImportConfStruct
	timeNow             string
	startTime           time.Time
	endTime             time.Duration
	espXmlmc            *apiLib.XmlmcInstStruct
	xmlmcInstanceConfig xmlmcConfigStruct
	mutex               = &sync.Mutex{}
	wg                  sync.WaitGroup
	wg2                 sync.WaitGroup
)

// ----- Structures -----
type counterTypeStruct struct {
	created        int
	createdSkipped int
}

//----- Config Data Structs
type swImportConfStruct struct {
	HBConf                    hbConfStruct //Hornbill Instance connection details
	SWServerAddress           string
	AttachmentRoot            string
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
	MethodResult string       `xml:"status,attr"`
	Params       paramsStruct `xml:"params"`
	State        stateStruct  `xml:"state"`
}

//----- Data Structs -----
//----- Site Structs
type siteListStruct struct {
	SiteName string
	SiteID   int
}
type xmlmcSiteListResponse struct {
	MethodResult string               `xml:"status,attr"`
	Params       paramsSiteListStruct `xml:"params"`
	State        stateStruct          `xml:"state"`
}
type paramsSiteListStruct struct {
	RowData paramsSiteRowDataListStruct `xml:"rowData"`
}
type paramsSiteRowDataListStruct struct {
	Row siteObjectStruct `xml:"row"`
}
type siteObjectStruct struct {
	SiteID      int    `xml:"h_id"`
	SiteName    string `xml:"h_site_name"`
	SiteCountry string `xml:"h_country"`
}

//----- Priority Structs
type priorityListStruct struct {
	PriorityName string
	PriorityID   int
}
type xmlmcPriorityListResponse struct {
	MethodResult string                   `xml:"status,attr"`
	Params       paramsPriorityListStruct `xml:"params"`
	State        stateStruct              `xml:"state"`
}
type paramsPriorityListStruct struct {
	RowData paramsPriorityRowDataListStruct `xml:"rowData"`
}
type paramsPriorityRowDataListStruct struct {
	Row priorityObjectStruct `xml:"row"`
}
type priorityObjectStruct struct {
	PriorityID   int    `xml:"h_pk_priorityid"`
	PriorityName string `xml:"h_priorityname"`
}

//----- Service Structs
type serviceListStruct struct {
	ServiceName string
	ServiceID   int
}
type xmlmcServiceListResponse struct {
	MethodResult string                  `xml:"status,attr"`
	Params       paramsServiceListStruct `xml:"params"`
	State        stateStruct             `xml:"state"`
}
type paramsServiceListStruct struct {
	RowData paramsServiceRowDataListStruct `xml:"rowData"`
}
type paramsServiceRowDataListStruct struct {
	Row serviceObjectStruct `xml:"row"`
}
type serviceObjectStruct struct {
	ServiceID   int    `xml:"h_pk_serviceid"`
	ServiceName string `xml:"h_servicename"`
}

//----- Team Structs
type teamListStruct struct {
	TeamName string
	TeamID   string
}
type xmlmcTeamListResponse struct {
	MethodResult string               `xml:"status,attr"`
	Params       paramsTeamListStruct `xml:"params"`
	State        stateStruct          `xml:"state"`
}
type paramsTeamListStruct struct {
	RowData paramsTeamRowDataListStruct `xml:"rowData"`
}
type paramsTeamRowDataListStruct struct {
	Row teamObjectStruct `xml:"row"`
}
type teamObjectStruct struct {
	TeamID   string `xml:"h_id"`
	TeamName string `xml:"h_name"`
}

//----- Category Structs
type categoryListStruct struct {
	CategoryCode string
	CategoryID   string
}
type xmlmcCategoryListResponse struct {
	MethodResult string               `xml:"status,attr"`
	Params       categoryObjectStruct `xml:"params"`
	State        stateStruct          `xml:"state"`
}
type categoryObjectStruct struct {
	CategoryID   string `xml:"id"`
	CategoryName string `xml:"fullname"`
}

//----- Audit Structs
type xmlmcAuditListResponse struct {
	MethodResult string            `xml:"status,attr"`
	Params       auditObjectStruct `xml:"params"`
	State        stateStruct       `xml:"state"`
}
type auditObjectStruct struct {
	TotalStorage     float64 `xml:"maxStorageAvailble"`
	TotalStorageUsed float64 `xml:"totalStorageUsed"`
}

//----- Analyst Structs
type analystListStruct struct {
	AnalystID   string
	AnalystName string
}
type xmlmcAnalystListResponse struct {
	MethodResult string                  `xml:"status,attr"`
	Params       paramsAnalystListStruct `xml:"params"`
	State        stateStruct             `xml:"state"`
}
type paramsAnalystListStruct struct {
	RowData paramsAnalystRowDataListStruct `xml:"rowData"`
}
type paramsAnalystRowDataListStruct struct {
	Row analystObjectStruct `xml:"row"`
}
type analystObjectStruct struct {
	AnalystID   string `xml:"h_user_id"`
	AnalystName string `xml:"h_name"`
}

//----- Associated Record Struct
type reqRelStruct struct {
	MasterRef string `db:"fk_callref_m"`
	SlaveRef  string `db:"fk_callref_s"`
}

//----- File Attachment Structs
type xmlmcAttachmentResponse struct {
	MethodResult string                 `xml:"status,attr"`
	Params       paramsAttachmentStruct `xml:"params"`
	State        stateStruct            `xml:"state"`
}
type paramsAttachmentStruct struct {
	ContentLocation string `xml:"contentLocation"`
}
type xmlmcNewAttachmentResponse struct {
	MethodResult string                    `xml:"status,attr"`
	Params       paramsNewAttachmentStruct `xml:"params"`
	State        stateStruct               `xml:"state"`
}
type paramsNewAttachmentStruct struct {
	EntityData entityDataStruct `xml:"primaryEntityData"`
}
type entityDataStruct struct {
	Record recordDataStruct `xml:"record"`
}
type recordDataStruct struct {
	PriKey string `xml:"h_pk_id"`
}

//----- Email Attachment Structs
type xmlmcEmailAttachmentResponse struct {
	MethodResult string                      `xml:"status,attr"`
	Params       paramsEmailAttachmentStruct `xml:"params"`
	State        stateStruct                 `xml:"state"`
}
type paramsEmailAttachmentStruct struct {
	Recipients []recipientStruct `xml:"recipient"`
	Subject    string            `xml:"subject"`
	Body       string            `xml:"body"`
	HTMLBody   string            `xml:"htmlBody"`
	TimeSent   string            `xml:"timeSent"`
}
type recipientStruct struct {
	Class   string `xml:"class"`
	Address string `xml:"address"`
	Name    string `xml:"name"`
}

//----- Shared Structs -----
type stateStruct struct {
	Code     string `xml:"code"`
	ErrorRet string `xml:"error"`
}
type paramsStruct struct {
	SessionID string `xml:"sessionId"`
	RequestID string `xml:"requestId"`
}

//----- File Attachment Struct
type fileAssocStruct struct {
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
	arrSWStatus["4"] = "status.open"
	arrSWStatus["5"] = "status.offHold"
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
	flag.Parse()

	//-- Output to CLI and Log
	logger(1, "---- Supportworks Call Import Utility V"+fmt.Sprintf("%v", version)+" ----", true)
	logger(1, "Flag - Config File "+fmt.Sprintf("%s", configFileName), true)
	logger(1, "Flag - Zone "+fmt.Sprintf("%s", configZone), true)
	logger(1, "Flag - Dry Run "+fmt.Sprintf("%v", configDryRun), true)

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
		processCallData()
	}
	//Process Service Requests
	mapGenericConf = swImportConf.ConfServiceRequest
	if mapGenericConf.Import == true {
		processCallData()
	}
	//Process Change Requests
	mapGenericConf = swImportConf.ConfChangeRequest
	if mapGenericConf.Import == true {
		processCallData()
	}
	//Process Problems
	mapGenericConf = swImportConf.ConfProblem
	if mapGenericConf.Import == true {
		processCallData()
	}
	//Process Known Errors
	mapGenericConf = swImportConf.ConfKnownError
	if mapGenericConf.Import == true {
		processCallData()
	}

	if len(arrCallsLogged) > 0 {
		//We have new calls logged - process associations
		processCallAssociations()
		//Process File Attachments
		processFileAttachments()
		//Close relevant Requests
		processClosedCalls()
	}

	//-- End output
	logger(1, "Requests Logged: "+fmt.Sprintf("%d", counters.created), true)
	logger(1, "Requests Skipped: "+fmt.Sprintf("%d", counters.createdSkipped), true)
	//-- Show Time Takens
	endTime = time.Now().Sub(startTime)
	logger(1, "Time Taken: "+fmt.Sprintf("%v", endTime), true)
	logger(1, "---- Supportworks Call Import Complete ---- ", true)
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
			// Add an entry to the request map
			currDataID := requestAttachments.DataID
			callLevel[currSMFileCallRef][currDataID] = requestAttachments
		}
	}

	_, _, strStorageTotal, strStorageAvailable := getInstanceFreeSpace()
	var fltStorageRequired float64
	//callLevel is a recurring map - now iterate through this to get file attachment size information
	for _, fileRecords := range callLevel {
		for _, fileRecord := range fileRecords {
			fltStorageRequired = fltStorageRequired + fileRecord.SizeU
		}
	}
	//intStorageRequired := int64(fltStorageRequired)
	strStorageRequired := convFloattoSizeStr(fltStorageRequired)

	logger(3, " ------------ File Attachment Processing ------------", true)
	logger(3, " Approximately "+strStorageRequired+" of storage space is required to import your", true)
	logger(3, " Request File Attachments.", true)
	logger(3, " You have approximately "+strStorageAvailable+" available space, from your subscribed", true)
	logger(3, " amount of "+strStorageTotal+".", true)
	fmt.Printf(" Do you want to import your Supportworks Call File Attachments\n in to your Service Manager Requests (yes/no): ")

	if confirmResponse() == true {
		//Iterate through File Attachment records again for processing
		logger(3, " Processing attachments for "+fmt.Sprintf("%v", len(callLevel))+" requests...", true)
		bar := pb.StartNew(len(callLevel))
		for callKey, fileRecords := range callLevel {
			bar.Increment()
			for _, fileRecord := range fileRecords {
				//If file has .SWM extension, rename to .TXT
				entityRequest := ""
				fileExtension := filepath.Ext(fileRecord.FileName)
				fileName := fileRecord.FileName
				if fileExtension == ".swm" {
					fileName = strings.TrimSuffix(fileName, fileExtension) + ".txt"
				}
				if fileRecord.UpdateID == "999999999" {
					entityRequest = "Requests"
				} else {
					entityRequest = "RequestHistoricUpdateAttachments"
				}

				boolAddOK := addFileRecord(callKey, entityRequest, fileName, fileRecord)
				if boolAddOK == true {
					addFileContent(callKey, entityRequest, fileName, fileRecord)
				}
			}
		}
		bar.FinishPrint("Request File Attachment Processing Complete")
		logger(1, "Request File Attachment Processing Complete", false)
	} else {
		logger(1, "No file attachments will be imported.", true)
	}
}

//addFileRecord - given a Master Reference and a Slave Refernce, adds a call association record to Service Manager
func addFileRecord(smCallRef, entityName, fileName string, fileRecord fileAssocStruct) bool {
	if entityName == "RequestHistoricUpdateAttachments" {
		espXmlmc.SetParam("application", appServiceManager)
		espXmlmc.SetParam("entity", "RequestHistoricUpdateAttachments")
		espXmlmc.OpenElement("primaryEntityData")
		espXmlmc.OpenElement("record")
		espXmlmc.SetParam("h_addedby", fileRecord.AddedBy)
		espXmlmc.SetParam("h_callref", smCallRef)
		espXmlmc.SetParam("h_compressed", fileRecord.Compressed)
		espXmlmc.SetParam("h_dataid", fileRecord.DataID)
		espXmlmc.SetParam("h_filename", fileName)
		espXmlmc.SetParam("h_filetime", fileRecord.FileTime)
		espXmlmc.SetParam("h_pk_fileid", fileRecord.FileID)
		espXmlmc.SetParam("h_sizec", strconv.Itoa(int(fileRecord.SizeC)))
		espXmlmc.SetParam("h_sizeu", strconv.Itoa(int(fileRecord.SizeU)))
		espXmlmc.SetParam("h_timeadded", fileRecord.TimeAdded)
		espXmlmc.SetParam("h_updateid", fileRecord.UpdateID)
		espXmlmc.CloseElement("record")
		espXmlmc.CloseElement("primaryEntityData")
		XMLHistAtt, xmlmcErr := espXmlmc.Invoke("data", "entityAddRecord")
		if xmlmcErr != nil {
			log.Fatal(xmlmcErr)
			return false
		}
		var xmlRespon xmlmcResponse
		errXMLMC := xml.Unmarshal([]byte(XMLHistAtt), &xmlRespon)
		if errXMLMC != nil {
			logger(4, "Unable to read response from Hornbill instance for File Attachment Record Insertion:"+fmt.Sprintf("%v", errXMLMC), false)
			return false
		}
		if xmlRespon.MethodResult != "ok" {
			logger(3, "Unable to process File Attachment Record Insertion: "+xmlRespon.State.ErrorRet, false)
			return false
		}
	}
	return true
}

//decodeSWMFile - reads the email attachment from Supportworks, returns the content as a string
func decodeSWMFile(fileEncoded string) (string, string) {
	emailContent := ""
	subjectLine := ""
	//Decode SWM in to struct
	espXmlmc.SetParam("fileContent", fileEncoded)
	XMLEmailDecoded, xmlmcErrEmail := espXmlmc.Invoke("mail", "decodeCompositeMessage")
	if xmlmcErrEmail != nil {
		log.Fatal(xmlmcErrEmail)
		return emailContent, subjectLine
	}
	var xmlResponEmail xmlmcEmailAttachmentResponse
	errUnmarshall := xml.Unmarshal([]byte(XMLEmailDecoded), &xmlResponEmail)
	if errUnmarshall != nil {
		logger(4, "Unable to read XML response from Message Decode: "+fmt.Sprintf("%v", errUnmarshall), false)
		return emailContent, subjectLine
	}
	if xmlResponEmail.MethodResult != "ok" {
		logger(4, "Error returned from API for Message Decode: "+fmt.Sprintf("%v", xmlResponEmail.MethodResult), false)
		return emailContent, subjectLine
	}

	if xmlResponEmail.Params.Recipients == nil {
		logger(4, "No recipients found in mail message.", false)
		return emailContent, subjectLine
	}

	//Build string to write to text file
	fromAddress := ""
	toAddress := ""
	for _, recipient := range xmlResponEmail.Params.Recipients {
		if recipient.Class == "from" {
			fromAddress = recipient.Address
		}
		if recipient.Class == "to" {
			toAddress = recipient.Address
		}
	}
	bodyText := ""
	if xmlResponEmail.Params.Body != "" {
		bodyText = xmlResponEmail.Params.Body
	} else {
		bodyText = xmlResponEmail.Params.HTMLBody
	}
	subjectLine = "Subject: " + xmlResponEmail.Params.Subject
	emailContent = "From: " + fromAddress + "\r\n"
	emailContent = emailContent + "To: " + toAddress + "\r\n"
	if xmlResponEmail.Params.TimeSent != "" {
		emailContent = emailContent + "Sent: " + epochToDateTime(xmlResponEmail.Params.TimeSent) + "\r\n"
	}
	emailContent = emailContent + subjectLine + "\r\n"
	emailContent = emailContent + strings.Repeat("-", len(subjectLine)) + "\r\n"
	emailContent = emailContent + bodyText
	return emailContent, subjectLine
}

//addFileContent - reads the file attachment from Supportworks, attach to request and update content location
func addFileContent(smCallRef, entityName, fileName string, fileRecord fileAssocStruct) bool {
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
		attPriKey = smCallRef
	}

	fileExtension := filepath.Ext(fileRecord.FileName)
	swmDecoded := ""
	subjectLine := ""
	if fileExtension == ".swm" {
		//Further processing for SWM files
		//Copy content in to TXT file, and attach this instead
		swmDecoded, subjectLine = decodeSWMFile(fileEncoded)
		if swmDecoded != "" {
			fileEncoded = base64.StdEncoding.EncodeToString([]byte(swmDecoded))
		}
	}

	//File content read - add data to instance
	espXmlmc.SetParam("application", appServiceManager)
	espXmlmc.SetParam("entity", entityName)
	espXmlmc.SetParam("keyValue", attPriKey)
	espXmlmc.SetParam("folder", "/")
	espXmlmc.OpenElement("localFile")
	espXmlmc.SetParam("fileName", fileName)
	espXmlmc.SetParam("fileData", fileEncoded)
	espXmlmc.CloseElement("localFile")
	espXmlmc.SetParam("overwrite", "true")
	XMLAttach, xmlmcErr := espXmlmc.Invoke("data", "entityAttachFile")
	if xmlmcErr != nil {
		logger(5, "Could not add Attachment File Data: "+fmt.Sprintf("%v", xmlmcErr), false)
		log.Fatal(xmlmcErr)
		return false
	}
	var xmlRespon xmlmcAttachmentResponse

	err := xml.Unmarshal([]byte(XMLAttach), &xmlRespon)
	if err != nil {
		logger(5, "Could not add Attachment File Data: "+fmt.Sprintf("%v", err), false)
	} else {
		if xmlRespon.MethodResult != "ok" {
			logger(5, "Could not add Attachment File Data: "+xmlRespon.State.ErrorRet, false)
		} else {
			//-- If we've got a Content Location back from the API, update the file record with this
			if xmlRespon.Params.ContentLocation != "" {
				strService := ""
				strMethod := ""
				if entityName == "RequestHistoricUpdateAttachments" {
					espXmlmc.SetParam("application", appServiceManager)
					espXmlmc.SetParam("entity", "RequestHistoricUpdateAttachments")
					espXmlmc.OpenElement("primaryEntityData")
					espXmlmc.OpenElement("record")
					espXmlmc.SetParam("h_pk_fileid", fileRecord.FileID)
					espXmlmc.SetParam("h_contentlocation", xmlRespon.Params.ContentLocation)
					espXmlmc.CloseElement("record")
					espXmlmc.CloseElement("primaryEntityData")
					strService = "data"
					strMethod = "entityUpdateRecord"
				} else {
					espXmlmc.SetParam("requestId", smCallRef)
					espXmlmc.SetParam("fileName", fileName)
					espXmlmc.SetParam("fileSource", xmlRespon.Params.ContentLocation)
					if subjectLine != "" {
						espXmlmc.SetParam("description", subjectLine+" - Originally added by "+fileRecord.AddedBy)
					} else {
						espXmlmc.SetParam("description", "Originally added by "+fileRecord.AddedBy)
					}
					espXmlmc.SetParam("visibility", "trustedGuest")
					strService = "apps/" + appServiceManager + "/Requests"
					strMethod = "attachFileFromServer"
				}
				XMLContentLoc, xmlmcErrContent := espXmlmc.Invoke(strService, strMethod)
				if xmlmcErrContent != nil {
					log.Fatal(xmlmcErrContent)
				}
				var xmlResponLoc xmlmcResponse

				err := xml.Unmarshal([]byte(XMLContentLoc), &xmlResponLoc)
				if err != nil {
					logger(4, "Added file data but unable to set Content Location for File Content - read response from Hornbill instance:"+fmt.Sprintf("%v", err), false)
					return false
				}
				if xmlResponLoc.MethodResult != "ok" {
					logger(4, "Added file data but unable to set Content Location for File Content: "+xmlResponLoc.State.ErrorRet, false)
					return false
				}
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
		fmt.Println("Please enter yes or no to continue:")
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
		logger(5, "Could not return Instance Audit Information: "+fmt.Sprintf("%v", xmlmcErr), false)
		log.Fatal(xmlmcErr)
	}
	var xmlRespon xmlmcAuditListResponse

	err := xml.Unmarshal([]byte(XMLAudit), &xmlRespon)
	if err != nil {
		logger(5, "Could not return Instance Audit Information: "+fmt.Sprintf("%v", err), false)
	} else {
		if xmlRespon.MethodResult != "ok" {
			logger(5, "Could not return Instance Audit Information: "+xmlRespon.State.ErrorRet, false)
		} else {
			//-- Check Response
			if xmlRespon.Params.TotalStorage > 0 && xmlRespon.Params.TotalStorageUsed > 0 {
				fltTotalSpace = xmlRespon.Params.TotalStorage
				fltFreeSpace = xmlRespon.Params.TotalStorage - xmlRespon.Params.TotalStorageUsed
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
	logger(3, "[DATABASE] Running query for Request Associations "+currentCallRef+". Please wait...", false)

	//build query
	sqlDiaryQuery := "SELECT fk_callref_m, fk_callref_s from cmn_rel_opencall_oc "
	logger(3, "[DATABASE} Request Association Query: "+sqlDiaryQuery, false)
	//Run Query
	rows, err := db.Queryx(sqlDiaryQuery)
	if err != nil {
		logger(4, " Database Query Error: "+fmt.Sprintf("%v", err), false)
		return
	}
	//Process each association record, insert in to Hornbill
	for rows.Next() {
		var requestRels reqRelStruct
		err = rows.StructScan(&requestRels)
		if err != nil {
			logger(4, " Data Mapping Error: "+fmt.Sprintf("%v", err), false)
			return
		}
		smMasterRef, mrOK := arrCallsLogged[requestRels.MasterRef]
		smSlaveRef, srOK := arrCallsLogged[requestRels.SlaveRef]

		if mrOK == true && smMasterRef != "" && srOK == true && smSlaveRef != "" {
			//We have Master and Slave calls matched in the SM database
			addAssocRecord(smMasterRef, smSlaveRef)
		}
	}
	logger(1, "Request Association Processing Complete", true)
}

//addAssocRecord - given a Master Reference and a Slave Refernce, adds a call association record to Service Manager
func addAssocRecord(masterRef, slaveRef string) {
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
		log.Fatal(xmlmcErr)
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
func processCallData() bool {
	boolCallDataSuccess := false
	if queryDBCallDetails(mapGenericConf.CallClass, connStrAppDB) == true {
		bar := pb.StartNew(len(arrCallDetailsMaps))
		//We have Call Details - insert them in to
		fmt.Println("Go Routines", maxGoroutines)
		maxGoroutinesGaurd := make(chan struct{}, maxGoroutines)
		for _, callRecord := range arrCallDetailsMaps {
			maxGoroutinesGaurd <- struct{}{}
			wg2.Add(1)
			arrCallRecord := callRecord
			callRecordCallref := callRecord["callref"]
			go func() {
				defer wg2.Done()
				time.Sleep(1 * time.Millisecond)
				mutex.Lock()
				bar.Increment()
				mutex.Unlock()
				callID := fmt.Sprintf("%s", callRecordCallref)
				currentCallRef = padCallRef(callID, "F", 7)
				boolCallLogged, hbCallRef := logNewCall(mapGenericConf.CallClass, arrCallRecord, callID)
				if boolCallLogged {
					logger(3, "[REQUEST LOGGED] Request logged successfully: "+hbCallRef+" from Supportworks call "+currentCallRef, false)
					boolCallDataSuccess = true
				} else {
					logger(4, mapGenericConf.CallClass+" call log failed: "+currentCallRef, false)
				}
				<-maxGoroutinesGaurd
			}()
		}
		wg2.Wait()

		bar.FinishPrint(mapGenericConf.CallClass + " Call Import Complete")
	} else {
		logger(4, "Call Search Failed for Call Class: "+mapGenericConf.CallClass, true)
	}
	return boolCallDataSuccess
}

//processClosedCalls - closes all relevant requests when logging process complete
// - This is to ensure that closed calls have file attachments associated, which cannot be done at point of logging
func processClosedCalls() {

	fmt.Println("Go Routines", maxGoroutines)
	maxGoroutinesGaurd := make(chan struct{}, maxGoroutines)

	for callref := range arrClosedCalls {
		maxGoroutinesGaurd <- struct{}{}
		wg.Add(1)
		callrefLocal := callref
		go func() {
			defer wg.Done()
			currSMFileCallRef, importedCallToClose := arrCallsLogged[callrefLocal]
			if importedCallToClose == true {
				//Close the current call!
				espXmlmc.SetParam("requestID", currSMFileCallRef)
				XMLClose, xmlmcErr := espXmlmc.Invoke("apps/"+appServiceManager+"/Requests", "closeRequest")
				if xmlmcErr != nil {
					log.Fatal(xmlmcErr)
					<-maxGoroutinesGaurd
					return
				}
				var xmlRespon xmlmcResponse
				err := xml.Unmarshal([]byte(XMLClose), &xmlRespon)
				if err != nil {
					logger(4, "Unable to read response from Hornbill instance to close request "+currSMFileCallRef+":"+fmt.Sprintf("%v", err), false)
					<-maxGoroutinesGaurd
					return
				}
				if xmlRespon.MethodResult != "ok" {
					logger(4, "Unable to close request "+currSMFileCallRef+": "+xmlRespon.State.ErrorRet, false)
					<-maxGoroutinesGaurd
					return
				}
			}

			<-maxGoroutinesGaurd
		}()
	}
	wg.Wait()
	return
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
	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return false, "Unable to create Session"
	}

	defer EspXmlmcSessionDestroy(espXmlmc)
	boolCallLoggedOK := false
	boolAssignToDefault := false
	strNewCallRef := ""
	//-- Get site ID
	siteID, siteName := getSiteID(callMap)
	//-- Get Priority ID
	strPriorityID, strPriorityName := getCallPriorityID(callMap)
	if strPriorityID == "" && mapGenericConf.DefaultPriority != "" {
		strPriorityID = getPriorityID(mapGenericConf.DefaultPriority)
		strPriorityName = mapGenericConf.DefaultPriority
	}
	//-- Get Team ID
	strTeamID := getCallTeamID(callMap)
	if strTeamID == "" && mapGenericConf.DefaultTeam != "" {
		strTeamID = getTeamID(mapGenericConf.DefaultTeam)
		if strTeamID != "" {
			boolAssignToDefault = true
		}
	}
	//-- Get Call Category ID
	strCategoryID := getCallCategoryID(callMap, "Request")

	//-- Get Owner ID
	strOwnerID := ""
	ownerMapping := fmt.Sprintf("%v", mapGenericConf.CoreFieldMapping["ownerId"])
	if ownerMapping != "" {
		strOwnerID = getFieldValue(ownerMapping, callMap)
	}
	//-- Get Service ID
	strServiceID := getCallServiceID(callMap)
	if strServiceID == "" && mapGenericConf.DefaultService != "" {
		strServiceID = getServiceID(mapGenericConf.DefaultService)
	}
	//-- Get Summary Text
	strSummary := ""
	summaryMapping := fmt.Sprintf("%v", mapGenericConf.CoreFieldMapping["summary"])
	if summaryMapping != "" {
		strSummary = getFieldValue(summaryMapping, callMap)
	}
	//-- Get Description Text
	strDescription := ""
	descriptionMapping := fmt.Sprintf("%v", mapGenericConf.CoreFieldMapping["description"])
	if descriptionMapping != "" {
		strDescription = getFieldValue(descriptionMapping, callMap)
	}
	//-- Get Request Status
	strStatus := ""
	statusMapping := fmt.Sprintf("%v", mapGenericConf.CoreFieldMapping["status"])
	if statusMapping != "" {
		if statusMapping == "16" || statusMapping == "18" {
			strStatus = arrSWStatus["6"]
			arrClosedCalls[swCallID] = statusMapping
		} else {
			strStatus = arrSWStatus[getFieldValue(statusMapping, callMap)]
		}
	}
	//-- Get Customer ID
	strCustID := ""
	custIDMapping := fmt.Sprintf("%v", mapGenericConf.CoreFieldMapping["customerId"])
	if custIDMapping != "" {
		strCustID = getFieldValue(custIDMapping, callMap)
	}
	//-- Get Customer Type
	strCustType := "1"
	custTypeMapping := fmt.Sprintf("%v", mapGenericConf.CoreFieldMapping["customerType"])
	if custTypeMapping != "" {
		strCustType = getFieldValue(custTypeMapping, callMap)
	}
	//-- Get Impact
	strImpact := ""
	impactMapping := fmt.Sprintf("%v", mapGenericConf.CoreFieldMapping["impact"])
	if impactMapping != "" {
		strImpact = getFieldValue(impactMapping, callMap)
	}
	//-- Get Urgency
	strUrgency := ""
	urgencyMapping := fmt.Sprintf("%v", mapGenericConf.CoreFieldMapping["urgency"])
	if urgencyMapping != "" {
		strUrgency = getFieldValue(urgencyMapping, callMap)
	}
	//-- Get Change Type
	strChangeType := ""
	if callClass == "Change Request" {
		changeTypeMapping := fmt.Sprintf("%v", mapGenericConf.CoreFieldMapping["changeType"])
		if changeTypeMapping != "" {
			strChangeType = getFieldValue(changeTypeMapping, callMap)
		}
	}

	//Set Params for new request log XMLMC call
	if strSummary != "" {
		espXmlmc.SetParam("summary", strSummary)
	}
	if strDescription != "" {
		espXmlmc.SetParam("description", strDescription)
	}
	espXmlmc.SetParam("requestType", callClass)
	if strCustID != "" {
		espXmlmc.SetParam("customerId", strCustID)
	}
	espXmlmc.SetParam("customerType", strCustType)
	if strStatus != "" {
		espXmlmc.SetParam("status", strStatus)
	}
	if strCategoryID != "" {
		espXmlmc.SetParam("categoryId", strCategoryID)
	}
	if strImpact != "" {
		espXmlmc.SetParam("impact", strImpact)
	}
	if strUrgency != "" {
		espXmlmc.SetParam("urgency", strUrgency)
	}
	if strServiceID != "" {
		espXmlmc.SetParam("serviceId", strServiceID)
	}
	if strChangeType != "" {
		espXmlmc.SetParam("changeType", strChangeType)
	}
	if siteID != "" && siteName != "" {
		espXmlmc.SetParam("siteId", siteID)
		espXmlmc.SetParam("siteName", siteName)
	}
	//-- Check for Dry Run
	if configDryRun != true {
		//Set the Service and Method for the XMLMC call
		strService := ""
		strMethod := ""
		switch callClass {
		case "Incident":
			strService = "Incidents"
			strMethod = "logIncident"
		case "Service Request":
			strService = "ServiceRequests"
			strMethod = "logServiceRequest"
		case "Change Request":
			strService = "ChangeRequests"
			strMethod = "logChangeRequest"
		case "Problem":
			strService = "Problems"
			strMethod = "logProblem"
		case "Known Error":
			strService = "KnownErrors"
			strMethod = "logKnownError"
		}

		XMLCreate, xmlmcErr := espXmlmc.Invoke("apps/"+appServiceManager+"/"+strService, strMethod)
		if xmlmcErr != nil {
			log.Fatal(xmlmcErr)
		}
		var xmlRespon xmlmcResponse

		err := xml.Unmarshal([]byte(XMLCreate), &xmlRespon)
		if err != nil {
			counters.createdSkipped++
			logger(4, "Unable to read response from Hornbill instance:"+fmt.Sprintf("%v", err), false)
			return false, "No"
		}
		if xmlRespon.MethodResult != "ok" {
			logger(4, "Unable to log request: "+xmlRespon.State.ErrorRet, false)
			counters.createdSkipped++
		} else {
			strNewCallRef = xmlRespon.Params.RequestID
			arrCallsLogged[swCallID] = strNewCallRef
			counters.created++
			boolCallLoggedOK = true
		}
	} else {
		//-- DEBUG XML TO LOG FILE
		var XMLSTRING = espXmlmc.GetParam()
		logger(1, "Request Log XML "+fmt.Sprintf("%s", XMLSTRING), false)
		counters.createdSkipped++
		espXmlmc.ClearParam()
		return true, "Dry Run"
	}

	//-- If request logged successfully :
	// Assign the Call to the Group/Analyst specified by the source call mapping or default value
	// Set the Priority on the call, from the source call mapping or default value
	// The above 2 are done outside of the original call logging, to prevent the possibility
	// of BPM Workflow errors
	//Then cycle through the additional call column mapping and update the parent call record
	//And finally get the Call Diary Updates from Supportworks and build the Historical Updates against the SM request
	if boolCallLoggedOK == true && strNewCallRef != "" {
		if strTeamID != "" {
			assignCall(strNewCallRef, strOwnerID, strTeamID, boolAssignToDefault)
		}
		if strPriorityID != "" {
			setPriority(strNewCallRef, strPriorityID, strPriorityName)
		}
		updateExtraRequestCols(strNewCallRef, callMap)
		applyHistoricalUpdates(strNewCallRef, swCallID)
	}
	return boolCallLoggedOK, strNewCallRef
}

//updateExtraRequestCols - takes additional mapping from config, updates call record accordingly
func updateExtraRequestCols(newCallRef string, callMap map[string]interface{}) bool {
	boolUpdateRequest := false
	strAttribute := ""
	strMapping := ""
	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return false
	}
	defer EspXmlmcSessionDestroy(espXmlmc)
	espXmlmc.SetParam("application", appServiceManager)
	espXmlmc.SetParam("entity", "Requests")
	espXmlmc.OpenElement("primaryEntityData")
	espXmlmc.OpenElement("record")
	espXmlmc.SetParam("h_pk_reference", newCallRef)
	for k, v := range mapGenericConf.AdditionalFieldMapping {
		strAttribute = fmt.Sprintf("%v", k)
		strMapping = fmt.Sprintf("%v", v)
		if strAttribute == "h_closure_category_id" && strMapping != "" {
			strClosureCategoryID := getCallCategoryID(callMap, "Closure")
			if strClosureCategoryID != "" {
				boolUpdateRequest = true
				espXmlmc.SetParam(strAttribute, strClosureCategoryID)
			}
		}
		if strMapping != "" && getFieldValue(strMapping, callMap) != "" {
			boolUpdateRequest = true
			espXmlmc.SetParam(strAttribute, getFieldValue(strMapping, callMap))
		}
	}

	if boolUpdateRequest == false {
		espXmlmc.ClearParam()
		return false
	}
	espXmlmc.CloseElement("record")
	espXmlmc.CloseElement("primaryEntityData")
	XMLCreate, xmlmcErr := espXmlmc.Invoke("data", "entityUpdateRecord")
	if xmlmcErr != nil {
		log.Fatal(xmlmcErr)
		return false
	}
	var xmlRespon xmlmcResponse

	err = xml.Unmarshal([]byte(XMLCreate), &xmlRespon)
	if err != nil {
		logger(4, "Unable to read response from Hornbill instance:"+fmt.Sprintf("%v", err), false)
		return false
	}
	if xmlRespon.MethodResult != "ok" {
		logger(4, "Unable to update request extended columns: "+xmlRespon.State.ErrorRet, false)
		return false
	}
	return true
}

//applyHistoricalUpdates - takes call diary records from Supportworks, imports to Hornbill as Historical Updates
func applyHistoricalUpdates(newCallRef, swCallRef string) bool {
	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return false
	}
	defer EspXmlmcSessionDestroy(espXmlmc)
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
	logger(3, "[DATABASE] Running query for Historical Updates of call "+currentCallRef+". Please wait...", false)

	//build query
	sqlDiaryQuery := "SELECT updatetimex, repid, groupid, udsource, udcode, udtype, updatetxt, udindex, timespent "
	sqlDiaryQuery = sqlDiaryQuery + " FROM updatedb WHERE callref = " + swCallRef + " ORDER BY udindex ASC"
	logger(3, "[DATABASE} Diary Query: "+sqlDiaryQuery, false)
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
			diaryTimex := fmt.Sprintf("%+s", diaryEntry["updatetimex"])
			diaryTime := epochToDateTime(diaryTimex)

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
			}

			espXmlmc.SetParam("application", appServiceManager)
			espXmlmc.SetParam("entity", "RequestHistoricUpdates")
			espXmlmc.OpenElement("primaryEntityData")
			espXmlmc.OpenElement("record")
			espXmlmc.SetParam("h_fk_reference", newCallRef)
			espXmlmc.SetParam("h_updatedate", diaryTime)
			intDiaryTimeSpent, _ := strconv.Atoi(fmt.Sprintf("%+s", diaryEntry["timespent"]))
			if intDiaryTimeSpent > 0 {
				espXmlmc.SetParam("h_timespent", fmt.Sprintf("%+s", diaryEntry["timespent"]))
			}
			if fmt.Sprintf("%+s", diaryEntry["udtype"]) != "" {
				espXmlmc.SetParam("h_updatetype", fmt.Sprintf("%+s", diaryEntry["udtype"]))
			}
			espXmlmc.SetParam("h_updatebytype", "1")
			espXmlmc.SetParam("h_updateindex", fmt.Sprintf("%+s", diaryEntry["udindex"]))
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
					log.Fatal(xmlmcErr)
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
				counters.createdSkipped++
				espXmlmc.ClearParam()
				return true
			}
		}
	}
	defer rows.Close()
	return true
}

//assignCall - takes Service Manager call ref, team and analyst, and assigns call accordingly
func assignCall(newCallRef, analystID, teamID string, boolDefaultAssign bool) (boolCallAssigned bool) {
	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return false
	}
	defer EspXmlmcSessionDestroy(espXmlmc)
	boolDoesOwnerExist := false
	if analystID != "" && boolDefaultAssign == false {
		boolDoesOwnerExist = doesOwnerExist(analystID)
	}
	espXmlmc.SetParam("inReference", newCallRef)
	if boolDoesOwnerExist == true {
		espXmlmc.SetParam("inAssignToId", analystID)
	}
	espXmlmc.SetParam("inAssignToGroupId", teamID)
	XMLCreate, xmlmcErr := espXmlmc.Invoke("apps/"+appServiceManager+"/Requests", "assign")
	if xmlmcErr != nil {
		log.Fatal(xmlmcErr)
		return
	}
	var xmlRespon xmlmcResponse
	err = xml.Unmarshal([]byte(XMLCreate), &xmlRespon)
	if err != nil {
		logger(4, "Unable to read response from Hornbill instance:"+fmt.Sprintf("%v", err), false)
		return
	}
	if xmlRespon.MethodResult != "ok" {
		logger(5, "Unable to assign request: "+xmlRespon.State.ErrorRet, false)
	} else {
		boolCallAssigned = true
	}
	return
}

//setPriority - takes Service Manager call ref and Priority ID, and Prioritises call accordingly
func setPriority(newCallRef, priorityID, priorityName string) (boolPrioritised bool) {
	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return false
	}
	defer EspXmlmcSessionDestroy(espXmlmc)
	espXmlmc.SetParam("requestId", newCallRef)
	espXmlmc.SetParam("priorityId", priorityID)
	espXmlmc.SetParam("priorityName", priorityName)
	espXmlmc.SetParam("escalationNote", "Priority "+priorityName+" assigned during import process.")
	XMLCreate, xmlmcErr := espXmlmc.Invoke("apps/"+appServiceManager+"/Requests", "escalateRequest")
	if xmlmcErr != nil {
		log.Fatal(xmlmcErr)
		return
	}
	var xmlRespon xmlmcResponse
	err = xml.Unmarshal([]byte(XMLCreate), &xmlRespon)
	if err != nil {
		logger(4, "Unable to read response from Hornbill instance:"+fmt.Sprintf("%v", err), false)
		return
	}
	if xmlRespon.MethodResult != "ok" {
		logger(5, "Unable to prioritise request: "+xmlRespon.State.ErrorRet, false)
	} else {
		boolPrioritised = true
	}
	return
}

// getFieldValue --Retrieve field value from mapping via SQL record map
func getFieldValue(v string, u map[string]interface{}) string {
	fieldMap := v
	//-- Match $variable from String
	re1, err := regexp.Compile(`\[(.*?)\]`)
	if err != nil {
		fmt.Printf("[ERROR] %v", err)
	}

	result := re1.FindAllString(fieldMap, 100)
	valFieldMap := ""
	//-- Loop Matches
	for _, val := range result {
		valFieldMap = ""
		valFieldMap = strings.Replace(val, "[", "", 1)
		valFieldMap = strings.Replace(valFieldMap, "]", "", 1)
		if valFieldMap == "oldCallRef" {
			fieldMap = strings.Replace(fieldMap, val, currentCallRef, 1)
		} else {
			if u[valFieldMap] != nil {
				valFieldMap = fmt.Sprintf("%+s", u[valFieldMap])
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
	siteNameMapping := fmt.Sprintf("%v", mapGenericConf.CoreFieldMapping["site"])
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
func getCallServiceID(callMap map[string]interface{}) string {
	serviceID := ""
	serviceNameMapping := fmt.Sprintf("%v", mapGenericConf.CoreFieldMapping["serviceId"])
	serviceName := getFieldValue(serviceNameMapping, callMap)
	if swImportConf.ServiceMapping[serviceName] != nil {
		serviceName = fmt.Sprintf("%s", swImportConf.ServiceMapping[serviceName])

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
func getCallPriorityID(callMap map[string]interface{}) (string, string) {
	priorityID := ""
	priorityNameMapping := fmt.Sprintf("%v", mapGenericConf.CoreFieldMapping["priorityId"])
	priorityName := getFieldValue(priorityNameMapping, callMap)
	if swImportConf.PriorityMapping[priorityName] != nil {
		priorityName = fmt.Sprintf("%s", swImportConf.PriorityMapping[priorityName])

		if priorityName != "" {
			priorityID = getPriorityID(priorityName)
		}
	}
	return priorityID, priorityName
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
func getCallTeamID(callMap map[string]interface{}) string {
	teamID := ""
	teamNameMapping := fmt.Sprintf("%v", mapGenericConf.CoreFieldMapping["teamId"])
	teamName := getFieldValue(teamNameMapping, callMap)
	if swImportConf.TeamMapping[teamName] != nil {
		teamName = fmt.Sprintf("%s", swImportConf.TeamMapping[teamName])
		if teamName != "" {
			teamID = getTeamID(teamName)
		}
	}
	return teamID
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
func getCallCategoryID(callMap map[string]interface{}, categoryGroup string) string {
	categoryID := ""
	categoryNameMapping := ""
	if categoryGroup == "Request" {
		categoryNameMapping = fmt.Sprintf("%v", mapGenericConf.CoreFieldMapping["categoryId"])
	} else {
		categoryNameMapping = fmt.Sprintf("%v", mapGenericConf.AdditionalFieldMapping["h_closure_category_id"])
	}
	categoryCode := getFieldValue(categoryNameMapping, callMap)

	if swImportConf.CategoryMapping[categoryCode] != nil {
		//Get Category Code from JSON mapping
		if swImportConf.CategoryMapping[categoryCode] != nil {
			categoryCode = fmt.Sprintf("%s", swImportConf.CategoryMapping[categoryCode])
		} else {
			//Mapping doesn't exist - replace hyphens from SW Profile code with colon, and try to use this
			categoryCode = strings.Replace(categoryCode, ":", "-", -1)
		}
	}
	if categoryCode != "" {
		categoryID = getCategoryID(categoryCode, categoryGroup)
	}
	return categoryID
}

//getCategoryID takes a Category Code string and returns a correct Category ID if one exists in the cache or on the Instance
func getCategoryID(categoryCode, categoryGroup string) string {
	categoryID := ""
	if categoryCode != "" {
		categoryIsInCache, CategoryIDCache := recordInCache(categoryCode, categoryGroup+"Category")
		//-- Check if we have cached the Category already
		if categoryIsInCache {
			categoryID = CategoryIDCache
		} else {
			categoryIsOnInstance, CategoryIDInstance := searchCategory(categoryCode, categoryGroup)
			//-- If Returned set output
			if categoryIsOnInstance {
				categoryID = CategoryIDInstance
			}
		}
	}
	return categoryID
}

//doesOwnerExist takes an Analyst ID string and returns a true if one exists in the cache or on the Instance
func doesOwnerExist(analystID string) bool {
	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return false
	}
	defer EspXmlmcSessionDestroy(espXmlmc)

	boolOwnerExists := false
	if analystID != "" {
		analystIsInCache, _ := recordInCache(analystID, "Analyst")
		//-- Check if we have cached the Analyst already
		if analystIsInCache {
			boolOwnerExists = true
		} else {
			//-- ESP Query for site
			espXmlmc.SetParam("application", appServiceManager)
			espXmlmc.SetParam("entity", "Colleagues")
			espXmlmc.SetParam("matchScope", "all")
			espXmlmc.OpenElement("searchFilter")
			espXmlmc.SetParam("h_user_id", analystID)
			espXmlmc.CloseElement("searchFilter")
			espXmlmc.SetParam("maxResults", "1")

			XMLAnalystSearch, xmlmcErr := espXmlmc.Invoke("data", "entityBrowseRecords")
			if xmlmcErr != nil {
				logger(5, "Unable to Search for Owner: "+fmt.Sprintf("%v", xmlmcErr), true)
				log.Fatal(xmlmcErr)
			}
			var xmlRespon xmlmcAnalystListResponse

			err := xml.Unmarshal([]byte(XMLAnalystSearch), &xmlRespon)
			if err != nil {
				logger(5, "Unable to Search for Owner: "+fmt.Sprintf("%v", err), true)
			} else {
				if xmlRespon.MethodResult != "ok" {
					logger(5, "Unable to Search for Owner: "+xmlRespon.State.ErrorRet, true)
				} else {
					//-- Check Response
					if xmlRespon.Params.RowData.Row.AnalystName != "" {
						boolOwnerExists = true
						//-- Add Priority to Cache
						var newAnalystForCache analystListStruct
						newAnalystForCache.AnalystID = analystID
						newAnalystForCache.AnalystName = xmlRespon.Params.RowData.Row.AnalystName
						name := []analystListStruct{newAnalystForCache}
						analysts = append(analysts, name...)
					}
				}
			}
		}
	}
	return boolOwnerExists
}

// recordInCache -- Function to check if passed-thorugh record name has been cached
// if so, pass back the Record ID
func recordInCache(recordName, recordType string) (bool, string) {
	boolReturn := false
	strReturn := ""
	switch recordType {
	case "Service":
		//-- Check if record in Priority Cache
		for _, service := range services {
			if service.ServiceName == recordName {
				boolReturn = true
				strReturn = strconv.Itoa(service.ServiceID)
			}
		}
	case "Priority":
		//-- Check if record in Priority Cache
		for _, priority := range priorities {
			if priority.PriorityName == recordName {
				boolReturn = true
				strReturn = strconv.Itoa(priority.PriorityID)
			}
		}
	case "Site":
		//-- Check if record in Site Cache
		for _, site := range sites {
			if site.SiteName == recordName {
				boolReturn = true
				strReturn = strconv.Itoa(site.SiteID)
			}
		}
	case "Team":
		//-- Check if record in Team Cache
		for _, team := range teams {
			if team.TeamName == recordName {
				boolReturn = true
				strReturn = team.TeamID
			}
		}
	case "Analyst":
		//-- Check if record in Analyst Cache
		for _, analyst := range analysts {
			if analyst.AnalystID == recordName {
				boolReturn = true
				strReturn = analyst.AnalystID
			}
		}
	case "RequestCategory":
		//-- Check if record in Category Cache
		for _, category := range categories {
			if category.CategoryCode == recordName {
				boolReturn = true
				strReturn = category.CategoryID
			}
		}
	case "ClosureCategory":
		//-- Check if record in Category Cache
		for _, category := range closeCategories {
			if category.CategoryCode == recordName {
				boolReturn = true
				strReturn = category.CategoryID
			}
		}
	}
	return boolReturn, strReturn
}

// seachSite -- Function to check if passed-through  site  name is on the instance
func searchSite(siteName string) (bool, int) {
	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return false, 0
	}
	defer EspXmlmcSessionDestroy(espXmlmc)

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
		logger(5, "Unable to Search for Site: "+fmt.Sprintf("%v", xmlmcErr), false)
		log.Fatal(xmlmcErr)
	}
	var xmlRespon xmlmcSiteListResponse

	err = xml.Unmarshal([]byte(XMLSiteSearch), &xmlRespon)
	if err != nil {
		logger(5, "Unable to Search for Site: "+fmt.Sprintf("%v", err), false)
	} else {
		if xmlRespon.MethodResult != "ok" {
			logger(5, "Unable to Search for Site: "+xmlRespon.State.ErrorRet, false)
		} else {
			//-- Check Response
			if xmlRespon.Params.RowData.Row.SiteName != "" {
				if strings.ToLower(xmlRespon.Params.RowData.Row.SiteName) == strings.ToLower(siteName) {
					intReturn = xmlRespon.Params.RowData.Row.SiteID
					boolReturn = true
					//-- Add Site to Cache
					var newSiteForCache siteListStruct
					newSiteForCache.SiteID = intReturn
					newSiteForCache.SiteName = siteName
					name := []siteListStruct{newSiteForCache}
					sites = append(sites, name...)
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
	defer EspXmlmcSessionDestroy(espXmlmc)
	espXmlmc.SetParam("application", appServiceManager)
	espXmlmc.SetParam("entity", "Priority")
	espXmlmc.SetParam("matchScope", "all")
	espXmlmc.OpenElement("searchFilter")
	espXmlmc.SetParam("h_priorityname", priorityName)
	espXmlmc.CloseElement("searchFilter")
	espXmlmc.SetParam("maxResults", "1")

	XMLPrioritySearch, xmlmcErr := espXmlmc.Invoke("data", "entityBrowseRecords")
	if xmlmcErr != nil {
		logger(5, "Unable to Search for Priority: "+fmt.Sprintf("%v", xmlmcErr), false)
		log.Fatal(xmlmcErr)
	}
	var xmlRespon xmlmcPriorityListResponse

	err = xml.Unmarshal([]byte(XMLPrioritySearch), &xmlRespon)
	if err != nil {
		logger(5, "Unable to Search for Priority: "+fmt.Sprintf("%v", err), false)
	} else {
		if xmlRespon.MethodResult != "ok" {
			logger(5, "Unable to Search for Priority: "+xmlRespon.State.ErrorRet, false)
		} else {
			//-- Check Response
			if xmlRespon.Params.RowData.Row.PriorityName != "" {
				if strings.ToLower(xmlRespon.Params.RowData.Row.PriorityName) == strings.ToLower(priorityName) {
					intReturn = xmlRespon.Params.RowData.Row.PriorityID
					boolReturn = true
					//-- Add Priority to Cache
					var newPriorityForCache priorityListStruct
					newPriorityForCache.PriorityID = intReturn
					newPriorityForCache.PriorityName = priorityName
					name := []priorityListStruct{newPriorityForCache}
					priorities = append(priorities, name...)
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
	defer EspXmlmcSessionDestroy(espXmlmc)
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
		logger(5, "Unable to Search for Service: "+fmt.Sprintf("%v", xmlmcErr), false)
		log.Fatal(xmlmcErr)
	}
	var xmlRespon xmlmcServiceListResponse

	err = xml.Unmarshal([]byte(XMLServiceSearch), &xmlRespon)
	if err != nil {
		logger(5, "Unable to Search for Service: "+fmt.Sprintf("%v", err), false)
	} else {
		if xmlRespon.MethodResult != "ok" {
			logger(5, "Unable to Search for Service: "+xmlRespon.State.ErrorRet, false)
		} else {
			//-- Check Response
			if xmlRespon.Params.RowData.Row.ServiceName != "" {
				if strings.ToLower(xmlRespon.Params.RowData.Row.ServiceName) == strings.ToLower(serviceName) {
					intReturn = xmlRespon.Params.RowData.Row.ServiceID
					boolReturn = true
					//-- Add Service to Cache
					var newServiceForCache serviceListStruct
					newServiceForCache.ServiceID = intReturn
					newServiceForCache.ServiceName = serviceName
					name := []serviceListStruct{newServiceForCache}
					services = append(services, name...)
				}
			}
		}
	}
	return boolReturn, intReturn
}

// seachTeam -- Function to check if passed-through support team name is on the instance
func searchTeam(teamName string) (bool, string) {
	boolReturn := false
	strReturn := ""
	//-- ESP Query for team
	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return false, "Unable to create connection"
	}
	defer EspXmlmcSessionDestroy(espXmlmc)
	espXmlmc.SetParam("userId", swImportConf.HBConf.UserName)
	espXmlmc.SetParam("password", base64.StdEncoding.EncodeToString([]byte(swImportConf.HBConf.Password)))
	espXmlmc.Invoke("session", "userLogon")
	espXmlmc.SetParam("entity", "Groups")
	espXmlmc.SetParam("matchScope", "all")
	espXmlmc.OpenElement("searchFilter")
	espXmlmc.SetParam("h_name", teamName)
	espXmlmc.CloseElement("searchFilter")
	espXmlmc.SetParam("maxResults", "1")

	XMLTeamSearch, xmlmcErr := espXmlmc.Invoke("data", "entityBrowseRecords")
	if xmlmcErr != nil {
		logger(5, "Unable to Search for Team: "+fmt.Sprintf("%v", xmlmcErr), true)
		log.Fatal(xmlmcErr)
	}
	var xmlRespon xmlmcTeamListResponse

	err = xml.Unmarshal([]byte(XMLTeamSearch), &xmlRespon)
	if err != nil {
		logger(5, "Unable to Search for Team: "+fmt.Sprintf("%v", err), true)
	} else {
		if xmlRespon.MethodResult != "ok" {
			logger(5, "Unable to Search for Team: "+xmlRespon.State.ErrorRet, true)
		} else {
			//-- Check Response
			if xmlRespon.Params.RowData.Row.TeamName != "" {
				if strings.ToLower(xmlRespon.Params.RowData.Row.TeamName) == strings.ToLower(teamName) {
					strReturn = xmlRespon.Params.RowData.Row.TeamID
					boolReturn = true
					//-- Add Team to Cache
					var newTeamForCache teamListStruct
					newTeamForCache.TeamID = strReturn
					newTeamForCache.TeamName = teamName
					name := []teamListStruct{newTeamForCache}
					teams = append(teams, name...)
				}
			}
		}
	}
	return boolReturn, strReturn
}

// seachCategory -- Function to check if passed-through support category name is on the instance
func searchCategory(categoryCode, categoryGroup string) (bool, string) {
	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return false, "Unable to create connection"
	}
	defer EspXmlmcSessionDestroy(espXmlmc)
	boolReturn := false
	strReturn := ""
	//-- ESP Query for category
	espXmlmc.SetParam("codeGroup", categoryGroup)
	espXmlmc.SetParam("code", categoryCode)

	XMLCategorySearch, xmlmcErr := espXmlmc.Invoke("data", "profileCodeLookup")
	if xmlmcErr != nil {
		logger(5, "Unable to Search for "+categoryGroup+" Category: "+fmt.Sprintf("%v", xmlmcErr), false)
		log.Fatal(xmlmcErr)
	}
	var xmlRespon xmlmcCategoryListResponse

	err = xml.Unmarshal([]byte(XMLCategorySearch), &xmlRespon)
	if err != nil {
		logger(5, "Unable to Search for "+categoryGroup+" Category: "+fmt.Sprintf("%v", err), false)
	} else {
		if xmlRespon.MethodResult != "ok" {
			logger(5, "Unable to Search for "+categoryGroup+" Category: "+xmlRespon.State.ErrorRet, false)
		} else {
			//-- Check Response
			if xmlRespon.Params.CategoryName != "" {
				strReturn = xmlRespon.Params.CategoryID
				boolReturn = true
				//-- Add Category to Cache
				var newCategoryForCache categoryListStruct
				newCategoryForCache.CategoryID = strReturn
				newCategoryForCache.CategoryCode = categoryCode
				name := []categoryListStruct{newCategoryForCache}
				switch categoryGroup {
				case "Request":
					categories = append(categories, name...)
				case "Closure":
					closeCategories = append(closeCategories, name...)
				}
			}
		}
	}
	return boolReturn, strReturn
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
			fmt.Printf("Error Creating Log Folder %q: %s \r", logPath, err)
			os.Exit(101)
		}
	}

	//-- Open Log File
	f, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0777)
	if err != nil {
		fmt.Printf("Error Creating Log File %q: %s \n", logFileName, err)
		os.Exit(100)
	}
	// don't forget to close it
	defer f.Close()
	// assign it to the standard logger
	log.SetOutput(f)
	var errorLogPrefix string
	//-- Create Log Entry
	switch t {
	case 1:
		errorLogPrefix = "[DEBUG] "
	case 2:
		errorLogPrefix = "[MESSAGE] "
	case 4:
		errorLogPrefix = "[ERROR] "
	case 5:
		errorLogPrefix = "[WARNING]"
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

//NewEspXmlmcSession - Creates new XMLMC Session for seperate instance initialisation
func NewEspXmlmcSession() (*apiLib.XmlmcInstStruct, error) {
	espXmlmcLocal := apiLib.NewXmlmcInstance(swImportConf.HBConf.URL)
	espXmlmcLocal.SetParam("userId", swImportConf.HBConf.UserName)
	espXmlmcLocal.SetParam("password", base64.StdEncoding.EncodeToString([]byte(swImportConf.HBConf.Password)))
	_, returncode := espXmlmcLocal.Invoke("session", "userLogon")

	return espXmlmcLocal, returncode
}

//EspXmlmcSessionDestroy - ends a given XMLMC session
func EspXmlmcSessionDestroy(XMLMCSession *apiLib.XmlmcInstStruct) {

	XMLMCSession.Invoke("session", "userLogoff")

}
