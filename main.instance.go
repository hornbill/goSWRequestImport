package main

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"log"
	"time"

	"github.com/hornbill/goapiLib"
)

// espLogger -- Log to ESP
func espLogger(message, severity string) {
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
				logger(4, "Unable to Search for Request Owner ["+analystID+"]: "+fmt.Sprintf("%v", xmlmcErr), false)
			}

			var xmlRespon xmlmcAnalystListResponse
			err := xml.Unmarshal([]byte(XMLAnalystSearch), &xmlRespon)
			if err != nil {
				logger(4, "Unable to Search for Request Owner ["+analystID+"]: "+fmt.Sprintf("%v", err), false)
			} else {
				if xmlRespon.MethodResult != "ok" {
					//Analyst most likely does not exist
					logger(5, "Unable to Search for Request Owner ["+analystID+"]: "+xmlRespon.State.ErrorRet, false)
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
	boolCustomerExists := false
	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return false
	}

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
				logger(4, "Unable to Search for Customer ["+customerID+"]: "+fmt.Sprintf("%v", xmlmcErr), false)
			}

			var xmlRespon xmlmcCustomerListResponse
			err := xml.Unmarshal([]byte(XMLCustomerSearch), &xmlRespon)
			if err != nil {
				logger(4, "Unable to Search for Customer ["+customerID+"]: "+fmt.Sprintf("%v", err), false)
			} else {
				if xmlRespon.MethodResult != "ok" {
					//Customer most likely does not exist
					logger(5, "Unable to Search for Customer ["+customerID+"]: "+xmlRespon.State.ErrorRet, false)
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

func checkInstanceSession() (string, bool) {
	errorMessage := ""
	boolValid := false
	espXmlmc := apiLib.NewXmlmcInstance(swImportConf.HBConf.URL)
	espXmlmc.SetSessionID(espXmlmc.GetSessionID())

	_, xmlmcErr := espXmlmc.Invoke("session", "isSessionValid")
	if xmlmcErr != nil {
		errorMessage = fmt.Sprintf("Unable to create new Hornbill session: %v", xmlmcErr)
	} else {
		boolValid = true
	}

	return errorMessage, boolValid
}

//NewEspXmlmcSession - New Xmlmc Session variable (Cloned Session)
func NewEspXmlmcSession() (*apiLib.XmlmcInstStruct, error) {
	time.Sleep(150 * time.Millisecond)
	espXmlmcLocal := apiLib.NewXmlmcInstance(swImportConf.HBConf.URL)
	espXmlmcLocal.SetSessionID(espXmlmc.GetSessionID())
	return espXmlmcLocal, nil
}

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
