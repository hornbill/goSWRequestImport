package main

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
)

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
func loggerGen(t int, s string) string {

	var errorLogPrefix = ""
	//-- Create Log Entry
	switch t {
	case 1:
		errorLogPrefix = "[DEBUG] "
	case 2:
		errorLogPrefix = "[MESSAGE] "
	case 3:
		errorLogPrefix = ""
	case 4:
		errorLogPrefix = "[ERROR] "
	case 5:
		errorLogPrefix = "[WARNING] "
	}
	return errorLogPrefix + s + "\n\r"
}
func loggerWriteBuffer(s string) {
	if s != "" {
		logLines := strings.Split(s, "\n\r")
		for _, line := range logLines {
			if line != "" {
				logger(0, line, false)
			}
		}
	}
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
	if err != nil {
		//We didnt manage to open the log file so exit the function
		return
	}
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
	case 0:
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
		errorLogPrefix = "[WARNING] "
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

//epochToDateTime - converts an EPOCH value STRING var in to a date-time format compatible with Hornbill APIs
func epochToDateTime(epochDateString string) string {
	dateTime := ""
	i, err := strconv.ParseInt(epochDateString, 10, 64)
	if err != nil {
		logger(5, "EPOCH String to Int conversion FAILED: "+fmt.Sprintf("%v", err), false)
	} else {
		dateTimeStr := time.Unix(i, 0).UTC().String() //Force UTC
		for i := 0; i < 19; i++ {
			dateTime = dateTime + string(dateTimeStr[i])
		}
	}
	return dateTime
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

//convExtendedColName - takes old extended column name, returns new one (supply h_custom_a returns h_custom_1 for example)
//Split string in to array with _ as seperator
//Convert last array entry string character to Rune
//Convert Rune to Integer
//Subtract 96 from Integer
//Convert resulting Integer to String (numeric character), append to prefix and pass back
func convExtendedColName(oldColName string) string {
	arrColName := strings.Split(oldColName, "_")
	strNewColID := strconv.Itoa(int([]rune(arrColName[2])[0]) - 96)
	return "h_custom_" + strNewColID
}

func getSupportworksIntRef(swCallRef string) int {
	var returnRef int
	re := regexp.MustCompile("[0-9]{1,}")
	intRef := re.FindAllString(swCallRef, 1)
	if intRef != nil {
		intRefSlice := strings.TrimLeft(intRef[0], "0")
		intRefInt, err := strconv.ParseInt(intRefSlice, 10, 0)
		if err != nil {
			logger(4, "Could not convert Supportworks reference number: "+fmt.Sprintf("%v", err), false)
		} else {
			returnRef = int(intRefInt)
		}
	}
	return returnRef
}

//parseFlags - grabs and parses command line flags
func parseFlags() {
	flag.StringVar(&configFileName, "file", "conf.json", "Name of the configuration file to load")
	flag.BoolVar(&configDryRun, "dryrun", false, "Dump import XML to log instead of creating requests")
	flag.BoolVar(&configDebug, "debug", false, "Additional logging for debugging.")
	flag.StringVar(&configMaxRoutines, "concurrent", "1", "Maximum number of requests to import concurrently.")
	flag.BoolVar(&boolProcessAttachments, "attachments", false, "Import attachemnts without prompting.")
	flag.Parse()
}

//getRequestPrefix - gets and returns current maxResultsAllowed sys setting value
func getRequestPrefix(callclass string) string {
	espXmlmc, sessErr := NewEspXmlmcSession()
	if sessErr != nil {
		logger(4, "Unable to attach to XMLMC session to get Request Prefix. Using default ["+callclass+"].", false)
		return callclass
	}

	strSetting := ""
	callclass = strings.ToLower(callclass)
	switch callclass {
	case "incident":
		strSetting = "guest.app.requests.types.IN"
	case "service request":
		strSetting = "guest.app.requests.types.SR"
	case "change request":
		strSetting = "app.requests.types.CH"
	case "problem":
		strSetting = "app.requests.types.PM"
	case "known error":
		strSetting = "app.requests.types.KE"
	case "release":
		strSetting = "app.requests.types.RM"
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
