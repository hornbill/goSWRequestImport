package main

import (
	"encoding/xml"
	"fmt"
	"html"
	"strconv"

	"github.com/hornbill/goapiLib"
)

//applyHistoricalUpdates - takes call diary records from Supportworks, imports to Hornbill as Historical Updates
func applyHistoricalUpdates(newCallRef string, swCallRef string, espXmlmc *apiLib.XmlmcInstStruct) bool {
	//Connect to the JSON specified DB
	//Check connection is open
	err := dbapp.Ping()
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
	logger(3, "[DATABASE] Diary Query: "+sqlDiaryQuery, false)
	mutex.Unlock()
	//Run Query
	rows, err := dbapp.Queryx(sqlDiaryQuery)
	if err != nil {
		logger(4, " Database Query Error: "+fmt.Sprintf("%v", err), false)
		return false
	}
	defer rows.Close()

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
					logger(3, "API Invoke Failed Unable to add Historical Call Diary Update: "+fmt.Sprintf("%v", xmlmcErr), false)
				}
				var xmlRespon xmlmcResponse
				errXMLMC := xml.Unmarshal([]byte(XMLUpdate), &xmlRespon)
				if errXMLMC != nil {
					logger(4, "Unable to read response from Hornbill instance:"+fmt.Sprintf("%v", errXMLMC), false)
				}
				if xmlRespon.MethodResult != "ok" {
					logger(3, "API Call Failed Unable to add Historical Call Diary Update: "+xmlRespon.State.ErrorRet, false)
				}
			} else {
				//-- DEBUG XML TO LOG FILE
				var XMLSTRING = espXmlmc.GetParam()
				logger(1, "Request Historical Update XML "+XMLSTRING, false)
				counters.Lock()
				counters.createdSkipped++
				counters.Unlock()
				espXmlmc.ClearParam()
				return true
			}
		}
	}
	return true
}

//optimiseIndex - optimises the HornbillITSMHistoric index once the historic update imports have completed
func optimiseIndex(espXmlmc *apiLib.XmlmcInstStruct) {
	logger(1, "Optimising HornbillITSMHistoric index, please wait...", true)

	espXmlmc.SetParam("indexStorage", "HornbillITSMHistoria")
	XMLUpdate, xmlmcErr := espXmlmc.Invoke("indexer", "indexOptimize")
	if xmlmcErr != nil {
		//log.Fatal(xmlmcErr)
		logger(5, "Unable to optimise index: "+fmt.Sprintf("%v", xmlmcErr), true)
		return
	}
	var xmlRespon xmlmcResponse
	errXMLMC := xml.Unmarshal([]byte(XMLUpdate), &xmlRespon)
	if errXMLMC != nil {
		logger(5, "Unable to read response from Hornbill instance:"+fmt.Sprintf("%v", errXMLMC), true)
		return
	}
	if xmlRespon.MethodResult != "ok" {
		logger(5, "Unable to optimise index: "+xmlRespon.State.ErrorRet, true)
		return
	}
	logger(1, "HornbillITSMHistoric index optimised", true)
}

func checkHistoricIndex() {
	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return
	}
	XMLUpdate, xmlmcErr := espXmlmc.Invoke("indexer", "getIndexStoragesList")
	if xmlmcErr != nil {
		//log.Fatal(xmlmcErr)
		logger(5, "Unable to list indexes: "+fmt.Sprintf("%v", xmlmcErr), false)
	}
	var xmlRespon xmlmcIndexListResponse
	errXMLMC := xml.Unmarshal([]byte(XMLUpdate), &xmlRespon)
	if errXMLMC != nil {
		logger(5, "Unable to read index list response from Hornbill instance:"+fmt.Sprintf("%v", errXMLMC), false)
	}
	if xmlRespon.MethodResult != "ok" {
		logger(5, "Unable to list indexes: "+xmlRespon.State.ErrorRet, false)
	}

	for _, index := range xmlRespon.Indexes {
		if index == "HornbillITSMHistoric" {
			optimiseIndex(espXmlmc)
		}
	}
}
