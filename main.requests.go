package main

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	_ "github.com/hornbill/goapiLib"
	"github.com/hornbill/pb"
)

//processCallData - Query Supportworks call data, process accordingly
func processCallData() {
	if queryDBCallDetails(mapGenericConf.CallClass, mapGenericConf.SupportworksCallClass, connStrAppDB) == true {
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
				mutexBar.Lock()
				bar.Increment()
				mutexBar.Unlock()

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
					logger(4, mapGenericConf.CallClass+" call log failed: "+currentCallRef+" - "+hbCallRef, false)
				}
				<-maxGoroutinesGuard
			}()
		}
		wgRequest.Wait()

		bar.FinishPrint(mapGenericConf.CallClass + " Call Import Complete")
	} else {
		logger(4, "Call Search Failed for Call Class: "+mapGenericConf.CallClass+"["+mapGenericConf.SupportworksCallClass+"]", true)
	}
}

//logNewCall - Function takes Supportworks call data in a map, and logs to Hornbill
func logNewCall(callClass string, callMap map[string]interface{}, swCallID string) (bool, string) {

	boolCallLoggedOK := false
	strNewCallRef := ""
	strStatus := ""
	boolOnHoldRequest := false

	//Get request status from request & map
	statusMapping := fmt.Sprintf("%v", mapGenericConf.CoreFieldMapping["h_status"])
	strStatusID := getFieldValue(statusMapping, callMap)
	if swImportConf.StatusMapping[strStatusID] != nil {
		strStatus = fmt.Sprintf("%v", swImportConf.StatusMapping[strStatusID])
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

		// Request Status
		if strAttribute == "h_status" {
			if strStatus == "status.onHold" {
				strStatus = "status.open"
				boolOnHoldRequest = true
			}
			espXmlmc.SetParam(strAttribute, strStatus)
			boolAutoProcess = false
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
			strAttribute != "h_status" &&
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

	//Extended Data Insert
	espXmlmc.OpenElement("relatedEntityData")
	espXmlmc.SetParam("relationshipName", "Extended Information")
	espXmlmc.SetParam("entityAction", "insert")
	espXmlmc.OpenElement("record")
	espXmlmc.SetParam("h_request_type", callClass)
	strAttribute = ""
	strMapping = ""
	//Loop through AdditionalFieldMapping fields from config, add to XMLMC Params if not empty
	for k, v := range mapGenericConf.AdditionalFieldMapping {
		strAttribute = fmt.Sprintf("%v", k)
		strSubString := "h_custom_"
		if strings.Contains(strAttribute, strSubString) {
			strAttribute = convExtendedColName(strAttribute)
			strMapping = fmt.Sprintf("%v", v)
			if strMapping != "" && getFieldValue(strMapping, callMap) != "" {
				espXmlmc.SetParam(strAttribute, getFieldValue(strMapping, callMap))
			}
		}
	}

	espXmlmc.CloseElement("record")
	espXmlmc.CloseElement("relatedEntityData")

	//-- Check for Dry Run
	if configDryRun != true {
		//XMLRequest := espXmlmc.GetParam()
		XMLCreate, xmlmcErr := espXmlmc.Invoke("data", "entityAddRecord")
		if xmlmcErr != nil {
			return false, fmt.Sprintf("%v", xmlmcErr)
		}
		var xmlRespon xmlmcRequestResponseStruct

		err := xml.Unmarshal([]byte(XMLCreate), &xmlRespon)
		if err != nil {
			counters.Lock()
			counters.createdSkipped++
			counters.Unlock()
			return false, fmt.Sprintf("%v", err)
		}
		if xmlRespon.MethodResult != "ok" {
			counters.Lock()
			counters.createdSkipped++
			counters.Unlock()
			boolCallLoggedOK = false
			strNewCallRef = xmlRespon.State.ErrorRet
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
				logger(5, "Activity Stream Creation failed for Request ["+strNewCallRef+"]", false)
			} else {
				var xmlRespon xmlmcResponse
				err = xml.Unmarshal([]byte(fixed), &xmlRespon)
				if err != nil {
					logger(5, "Activity Stream Creation unmarshall failed for Request ["+strNewCallRef+"]", false)
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
				espXmlmc.ClearParam()
				espXmlmc.SetParam("application", appServiceManager)
				espXmlmc.SetParam("entity", "Requests")
				espXmlmc.OpenElement("primaryEntityData")
				espXmlmc.OpenElement("record")
				espXmlmc.SetParam("h_pk_reference", strNewCallRef)
				espXmlmc.SetParam("h_datelogged", strLoggedDate)
				espXmlmc.CloseElement("record")
				espXmlmc.CloseElement("primaryEntityData")
				XMLLogDate, xmlmcErr := espXmlmc.Invoke("data", "entityUpdateRecord")
				if xmlmcErr != nil {
					logger(4, "Unable to update Log Date of request ["+strNewCallRef+"] : "+fmt.Sprintf("%v", xmlmcErr), false)
				}
				var xmlRespon xmlmcResponse

				errLogDate := xml.Unmarshal([]byte(XMLLogDate), &xmlRespon)
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

				//logger(1, callClass+" Logged: "+strNewCallRef+". Open Request status, spawing BPM Process "+strServiceBPM, false)
				if strNewCallRef != "" && strServiceBPM != "" {
					espXmlmc.ClearParam()
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
						logger(4, "Unable to read response when invoking BPM for request ["+strNewCallRef+"]:"+fmt.Sprintf("%v", errBPM), false)
					}
					if xmlRespon.MethodResult != "ok" {
						logger(4, "Unable to invoke BPM for request ["+strNewCallRef+"]: "+xmlRespon.State.ErrorRet, false)
					} else {

						//time.Sleep(500 * time.Millisecond)
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
							logger(4, "Unable to read response from Hornbill instance when updating BPM on ["+strNewCallRef+"]:"+fmt.Sprintf("%v", errBPMSpawn), false)
						}
						if xmlRespon.MethodResult != "ok" {
							logger(4, "Unable to associate BPM to Request ["+strNewCallRef+"]: "+xmlRespon.State.ErrorRet, false)
						}
					}
				}
			}

			// Now handle calls in an On Hold status
			if boolOnHoldRequest {
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

			//Add file attachments to request
			processFileAttachments(swCallID, strNewCallRef)
		}
	} else {
		//-- DEBUG XML TO LOG FILE
		var XMLSTRING = espXmlmc.GetParam()
		logger(1, "Request Log XML "+XMLSTRING, false)
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
