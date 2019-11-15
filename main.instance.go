package main

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"

	/* non core libraries */
	apiLib "github.com/hornbill/goApiLib"
)

// espLogger -- Log to ESP
func espLogger(message, severity string) {
	espXmlmc.SetParam("fileName", "SW_Call_Import")
	espXmlmc.SetParam("group", "general")
	espXmlmc.SetParam("severity", severity)
	espXmlmc.SetParam("message", message)
	espXmlmc.Invoke("system", "logMessage")
}

//doesAnalystExist takes an Analyst ID string and returns a true if one exists in the cache or on the Instance
func doesAnalystExist(analystID string, espXmlmc *apiLib.XmlmcInstStruct, buffer *bytes.Buffer) bool {

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
				buffer.WriteString(loggerGen(4, "Unable to Search for Request Owner ["+analystID+"]: "+fmt.Sprintf("%v", xmlmcErr)))
			}

			var xmlRespon xmlmcAnalystListResponse
			err := xml.Unmarshal([]byte(XMLAnalystSearch), &xmlRespon)
			if err != nil {
				buffer.WriteString(loggerGen(4, "Unable to Search for Request Owner ["+analystID+"]: "+fmt.Sprintf("%v", err)))
			} else {
				if xmlRespon.MethodResult != "ok" {
					//Analyst most likely does not exist
					buffer.WriteString(loggerGen(5, "Unable to Search for Request Owner ["+analystID+"]: "+xmlRespon.State.ErrorRet))
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
func doesCustomerExist(customerID string, espXmlmc *apiLib.XmlmcInstStruct, buffer *bytes.Buffer) bool {
	boolCustomerExists := false

	if customerID != "" {
		customerIsInCache, strReturn := recordInCache(customerID, "Customer")
		//-- Check if we have cached the Analyst already
		if customerIsInCache && strReturn != "" {
			boolCustomerExists = true
		} else {
			//Get Analyst Info
			if swImportConf.CustomerType == "1" {
				espXmlmc.SetParam("entity", "Contact")
			} else {
				espXmlmc.SetParam("entity", "UserAccount")
			}
			espXmlmc.SetParam("matchScope", "all")

			espXmlmc.OpenElement("searchFilter")
			if swImportConf.CustomerType == "1" {
				espXmlmc.SetParam("column", "h_logon_id")
			} else {
				espXmlmc.SetParam("column", "h_user_id")
			}
			espXmlmc.SetParam("value", customerID)
			espXmlmc.SetParam("matchType", "exact")
			espXmlmc.CloseElement("searchFilter")
			espXmlmc.SetParam("maxResults", "1")
			//fmt.Println(espXmlmc.GetParam())
			//fmt.Println(swImportConf.CustomerType)

			XMLCustomerSearch, xmlmcErr := espXmlmc.Invoke("data", "entityBrowseRecords2")
			if xmlmcErr != nil {
				buffer.WriteString(loggerGen(4, "Unable to Search for Customer ["+customerID+"]: "+fmt.Sprintf("%v", xmlmcErr)))
			}
			//fmt.Println(XMLCustomerSearch)
			//xmlRespon := nil
			if swImportConf.CustomerType == "1" {
				var xmlRespon xmlmcContactListResponse

				err := xml.Unmarshal([]byte(XMLCustomerSearch), &xmlRespon)
				if err != nil {
					buffer.WriteString(loggerGen(4, "Unable to Search for Customer ["+customerID+"]: "+fmt.Sprintf("%v", err)))
				} else {
					if xmlRespon.MethodResult != "ok" {
						//Customer most likely does not exist
						buffer.WriteString(loggerGen(5, "Unable to Search for Customer ["+customerID+"]: "+xmlRespon.State.ErrorRet))
					} else {
						//-- Check Response
						if xmlRespon.CustomerFirstName != "" {
							boolCustomerExists = true
							//-- Add Customer to Cache
							var newCustomerForCache customerListStruct
							newCustomerForCache.CustomerID = customerID
							newCustomerForCache.CustomerHornbillID = xmlRespon.CustomerHornbillID
							newCustomerForCache.CustomerOrgID = xmlRespon.CustomerOrgID
							newCustomerForCache.CustomerName = xmlRespon.CustomerFirstName + " " + xmlRespon.CustomerLastName
							customerNamedMap := []customerListStruct{newCustomerForCache}
							mutexCustomers.Lock()
							customers = append(customers, customerNamedMap...)
							mutexCustomers.Unlock()

							buffer.WriteString(loggerGen(1, "Added Customer ["+customerID+"]: "+newCustomerForCache.CustomerName))

						}
					}
				}

			} else {
				var xmlRespon xmlmcCustomerListResponse

				err := xml.Unmarshal([]byte(XMLCustomerSearch), &xmlRespon)
				if err != nil {
					buffer.WriteString(loggerGen(4, "Unable to Search for Customer ["+customerID+"].: "+fmt.Sprintf("%v", err)))
				} else {
					if xmlRespon.MethodResult != "ok" {
						//Customer most likely does not exist
						buffer.WriteString(loggerGen(5, "Unable to Search for Customer ["+customerID+"].: "+xmlRespon.State.ErrorRet))
					} else {
						//-- Check Response
						if xmlRespon.CustomerFirstName != "" {
							boolCustomerExists = true
							//-- Add Customer to Cache
							var newCustomerForCache customerListStruct
							newCustomerForCache.CustomerID = customerID
							newCustomerForCache.CustomerOrgID = xmlRespon.CustomerOrgID
							newCustomerForCache.CustomerHornbillID = xmlRespon.CustomerHornbillID
							newCustomerForCache.CustomerName = xmlRespon.CustomerFirstName + " " + xmlRespon.CustomerLastName
							customerNamedMap := []customerListStruct{newCustomerForCache}
							mutexCustomers.Lock()
							customers = append(customers, customerNamedMap...)
							mutexCustomers.Unlock()

							buffer.WriteString(loggerGen(1, "Added Customer ["+customerID+"].: "+newCustomerForCache.CustomerName))

						}
					}
				}
			}
		}
	}
	return boolCustomerExists
}

//NewEspXmlmcSession - New Xmlmc Session variable (Cloned Session)
func NewEspXmlmcSession() (*apiLib.XmlmcInstStruct, error) {
	espXmlmcLocal := apiLib.NewXmlmcInstance(swImportConf.HBConf.InstanceID)
	espXmlmcLocal.SetSessionID(espXmlmc.GetSessionID())
	return espXmlmcLocal, nil
}

//-- start ESP user session
func login() bool {
	logger(1, "Logging Into: "+swImportConf.HBConf.InstanceID, true)
	logger(1, "UserName: "+swImportConf.HBConf.UserName, false)
	espXmlmc = apiLib.NewXmlmcInstance(swImportConf.HBConf.InstanceID)

	espXmlmc.SetParam("userId", swImportConf.HBConf.UserName)
	espXmlmc.SetParam("password", base64.StdEncoding.EncodeToString([]byte(swImportConf.HBConf.Password)))
	XMLLogin, xmlmcErr := espXmlmc.Invoke("session", "userLogon")
	if xmlmcErr != nil {
		logger(4, "Unable to Login: "+fmt.Sprintf("%v", xmlmcErr), true)
		return false
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
