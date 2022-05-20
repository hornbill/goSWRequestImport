package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	/* non core libraries */

	apiLib "github.com/hornbill/goApiLib"
	"github.com/hornbill/pb"
)

//processCallData - Query Supportworks call data, process accordingly
func processCallData() {
	if queryDBCallDetails(mapGenericConf.CallClass, mapGenericConf.SupportworksCallClass, connStrAppDB) {
		bar := pb.StartNew(len(arrCallDetailsMaps))

		var wg sync.WaitGroup

		jobs := make(chan RequestDetails, maxGoroutines)

		for w := 1; w <= maxGoroutines; w++ {
			wg.Add(1)
			espXmlmc, err := NewEspXmlmcSession()
			if err != nil {
				logger(4, "Could not connect to Hornbill Instance: "+err.Error(), true)
				os.Exit(1)
			}
			go logNewCall(jobs, &wg, espXmlmc)
		}

		for _, callRecord := range arrCallDetailsMaps {
			callRecordArr := callRecord
			callRecordCallref := callRecord["callref"]

			mutexBar.Lock()
			bar.Increment()
			mutexBar.Unlock()

			callID := ""
			if callInt, ok := callRecordCallref.(int64); ok {
				callID = strconv.FormatInt(callInt, 10)
			} else {
				callID = fmt.Sprintf("%s", callRecordCallref)
			}
			jobs <- RequestDetails{CallClass: mapGenericConf.CallClass, CallMap: callRecordArr, SwCallID: callID}
		}

		close(jobs)
		wg.Wait()

		bar.FinishPrint(mapGenericConf.CallClass + " Call Import Complete")
	} else {
		logger(4, "Call Search Failed for Call Class: "+mapGenericConf.CallClass+"["+mapGenericConf.SupportworksCallClass+"]", true)
	}
}

//logNewCall - Function takes Supportworks call data in a map, and logs to Hornbill
func logNewCall(jobs chan RequestDetails, wg *sync.WaitGroup, espXmlmc *apiLib.XmlmcInstStruct) {
	defer wg.Done()
	for requestRecord := range jobs {

		var buffer bytes.Buffer

		callClass := requestRecord.CallClass
		callMap := requestRecord.CallMap
		swCallID := requestRecord.SwCallID
		buffer.WriteString(loggerGen(3, "   "))
		buffer.WriteString(loggerGen(1, "Buffer For Supportworks Ref: "+swCallID))

		if smMappedRef, ok := swImportConf.ExistingRequestMappings[swCallID]; ok {
			request := RequestReferences{SwCallID: swCallID, SmCallID: smMappedRef}
			applyHistoricalUpdates(request, espXmlmc, &buffer)
			mutexArrCallsLogged.Lock()
			arrCallsLogged[swCallID] = smMappedRef
			mutexArrCallsLogged.Unlock()
			mutexCounters.Lock()
			counters.existingRequests++
			mutexCounters.Unlock()
			continue
		}

		strNewCallRef := ""
		strStatus := ""
		boolOnHoldRequest := false

		//Get request status from request & map
		statusMapping := fmt.Sprintf("%v", mapGenericConf.CoreFieldMapping["h_status"])
		strStatusID := getFieldValue(statusMapping, callMap)
		if swImportConf.StatusMapping[strStatusID] != nil {
			strStatus = strings.ToLower(fmt.Sprintf("%v", swImportConf.StatusMapping[strStatusID]))
		}

		coreFields := make(map[string]string)
		strAttribute := ""
		strMapping := ""
		strServiceBPM := ""

		boolUpdateLogDate := false
		strLoggedDate := ""
		//Sort out logged date
		if logDateInterface, ok := mapGenericConf.CoreFieldMapping["h_datelogged"]; ok {
			if logDateInterface != "" {
				logDateMapping := fmt.Sprint(mapGenericConf.CoreFieldMapping["h_datelogged"])
				loggedEPOCH := getFieldValue(logDateMapping, callMap)
				if loggedEPOCH != "" && loggedEPOCH != "0" {
					strLoggedDate = epochToDateTime(loggedEPOCH)
					boolUpdateLogDate = true
				}
			}
		}

		//Sort out closed date
		strClosedDate := ""
		if closeDateInterface, ok := mapGenericConf.CoreFieldMapping["h_dateclosed"]; ok {
			if closeDateInterface != "" {
				closeDateMapping := fmt.Sprint(mapGenericConf.CoreFieldMapping["h_dateclosed"])
				closedEPOCH := getFieldValue(closeDateMapping, callMap)
				if closedEPOCH != "" && closedEPOCH != "0" {
					strClosedDate = epochToDateTime(closedEPOCH)
				}
			}
		}
		//Loop through core fields from config, add to XMLMC Params
		for k, v := range mapGenericConf.CoreFieldMapping {
			boolAutoProcess := true
			strAttribute = fmt.Sprintf("%v", k)
			strMapping = fmt.Sprintf("%v", v)

			if configCustomerOrg && (strAttribute == "h_org_id" || strAttribute == "h_company_id" || strAttribute == "h_company_name") {
				//Taking customer org/company from contact organisation or user home group - so do nothing with exiting mapping
				continue
			}
			//Owning Analyst Name
			if strAttribute == "h_ownerid" {
				strOwnerID := getFieldValue(strMapping, callMap)
				if strOwnerID != "" {
					boolAnalystExists := doesUserExist(strOwnerID, espXmlmc, &buffer)
					if boolAnalystExists {
						//Get analyst from cache as exists
						analystIsInCache, strOwnerName, _ := userInCache(strOwnerID)
						if analystIsInCache && strOwnerName != "" {
							coreFields[strAttribute] = strOwnerID
							coreFields["h_ownername"] = strOwnerName
						}
					}
				}
				boolAutoProcess = false
			}

			//Customer ID & Name
			if strAttribute == "h_fk_user_id" {
				strCustID := getFieldValue(strMapping, callMap)
				if strCustID != "" {
					if swImportConf.CustomerType == "1" {
						//Customer is a Contact
						contactExists := doesContactExist(strCustID, espXmlmc, &buffer)
						if contactExists {
							contactInCache, contactName, contactPK, contactOrgID := contactInCache(strCustID)
							if contactInCache && contactName != "" && contactPK != "" {
								coreFields[strAttribute] = contactPK
								coreFields["h_fk_user_name"] = contactName
								if configCustomerOrg && contactOrgID != "" {
									//Now sort out container
									coreFields["h_org_id"] = contactOrgID
									foundOrg, OrgContainerID := recordInCache(contactOrgID, "Organisation")
									if foundOrg && OrgContainerID != "" {
										coreFields["h_container_id"] = OrgContainerID
									}
								}
							}
						}
					} else {
						//Customer is a User
						boolCustExists := doesUserExist(strCustID, espXmlmc, &buffer)
						if boolCustExists {
							//Get customer from cache as exists
							customerIsInCache, strCustName, homeOrgID := userInCache(strCustID)
							if customerIsInCache {
								coreFields[strAttribute] = strCustID
								coreFields["h_fk_user_name"] = strCustName
								if configCustomerOrg && homeOrgID != "" {

									companyFound, companyName := searchGroup(homeOrgID, espXmlmc, &buffer)
									if companyFound {
										coreFields["h_company_id"] = homeOrgID
										coreFields["h_company_name"] = companyName
									}
								}
							}
						}
					}

				}
				boolAutoProcess = false
			}

			//Priority ID & Name
			//-- Get Priority ID
			if strAttribute == "h_fk_priorityid" {
				strPriorityID := getFieldValue(strMapping, callMap)
				strPriorityMapped, strPriorityName := getCallPriorityID(strPriorityID, espXmlmc, &buffer)
				if strPriorityMapped == "" && mapGenericConf.DefaultPriority != "" {
					strPriorityMapped = getPriorityID(mapGenericConf.DefaultPriority, espXmlmc, &buffer)
					strPriorityName = mapGenericConf.DefaultPriority
				}
				coreFields[strAttribute] = strPriorityMapped
				coreFields["h_fk_priorityname"] = strPriorityName
				boolAutoProcess = false
			}

			// Category ID & Name
			if strAttribute == "h_category_id" && strMapping != "" {
				//-- Get Call Category ID
				strCategoryID, strCategoryName := getCallCategoryID(callMap, "Request", espXmlmc, &buffer)
				if strCategoryID != "" && strCategoryName != "" {
					coreFields[strAttribute] = strCategoryID
					coreFields["h_category"] = strCategoryName
				}
				boolAutoProcess = false
			}

			// Closure Category ID & Name
			if strAttribute == "h_closure_category_id" && strMapping != "" {
				strClosureCategoryID, strClosureCategoryName := getCallCategoryID(callMap, "Closure", espXmlmc, &buffer)
				if strClosureCategoryID != "" {
					coreFields[strAttribute] = strClosureCategoryID
					coreFields["h_closure_category"] = strClosureCategoryName
				}
				boolAutoProcess = false
			}

			// Service ID & Name, & BPM Workflow
			if strAttribute == "h_fk_serviceid" {
				//-- Get Service ID
				swServiceID := getFieldValue(strMapping, callMap)
				strServiceID := getCallServiceID(swServiceID, espXmlmc, &buffer)
				if strServiceID == "" && mapGenericConf.DefaultService != "" {
					strServiceID = getServiceID(mapGenericConf.DefaultService, espXmlmc, &buffer)
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
							case "Release":
								strServiceBPM = service.ServiceBPMRelease
							}
						}
					}
					mutexServices.Unlock()

					if strServiceName != "" {
						coreFields[strAttribute] = strServiceID
						coreFields["h_fk_servicename"] = strServiceName
					}
				}
				boolAutoProcess = false
			}

			// Team ID and Name
			if strAttribute == "h_fk_team_id" {
				//-- Get Team ID
				swTeamID := getFieldValue(strMapping, callMap)
				strTeamID, strTeamName := getCallTeamID(swTeamID, espXmlmc, &buffer)
				if strTeamID == "" && mapGenericConf.DefaultTeam != "" {
					strTeamName = mapGenericConf.DefaultTeam
					strTeamID = getTeamID(strTeamName, espXmlmc, &buffer)
				}
				if strTeamID != "" && strTeamName != "" {
					coreFields[strAttribute] = strTeamID
					coreFields["h_fk_team_name"] = strTeamName
				}
				boolAutoProcess = false
			}

			// Site ID and Name
			if strAttribute == "h_site" {
				//-- Get site ID
				siteID, siteName := getSiteID(callMap, espXmlmc, &buffer)
				if siteID != "" && siteName != "" {
					coreFields["h_site_id"] = siteID
					coreFields[strAttribute] = siteName
				}
				boolAutoProcess = false
			}

			// Resolved Date/Time
			if strAttribute == "h_dateresolved" && strMapping != "" && (strStatus == "status.resolved" || strStatus == "status.closed") {
				resolvedEPOCH := getFieldValue(strMapping, callMap)
				if resolvedEPOCH != "" && resolvedEPOCH != "0" {
					strResolvedDate := epochToDateTime(resolvedEPOCH)
					if strResolvedDate != "" {
						coreFields[strAttribute] = strResolvedDate
					}
				}
			}

			// Closed Date/Time
			if strAttribute == "h_dateclosed" && strClosedDate != "" && strStatus != "status.onhold" {
				coreFields[strAttribute] = strClosedDate
			}

			// Request Status
			if strAttribute == "h_status" {
				if strStatus == "status.onhold" {
					strStatus = "status.open"
					boolOnHoldRequest = true
				}
				if strStatus == "status.cancelled" {
					coreFields["h_archived"] = "1"
				}
				coreFields[strAttribute] = strStatus
				boolAutoProcess = false
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
				strAttribute != "h_site_id" &&
				strAttribute != "h_fk_priorityname" &&
				strAttribute != "h_ownername" &&
				strAttribute != "h_fk_user_name" &&
				strAttribute != "h_datelogged" &&
				strAttribute != "h_dateresolved" &&
				strAttribute != "h_dateclosed" {

				if strMapping != "" && getFieldValue(strMapping, callMap) != "" {
					coreFields[strAttribute] = getFieldValue(strMapping, callMap)
				}
			}

		}

		espXmlmc.SetParam("application", appServiceManager)
		espXmlmc.SetParam("entity", "Requests")
		espXmlmc.SetParam("returnModifiedData", "true")
		espXmlmc.OpenElement("primaryEntityData")
		espXmlmc.OpenElement("record")

		for k, v := range coreFields {
			espXmlmc.SetParam(k, v)
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
		if !configDryRun {
			XMLRequest := espXmlmc.GetParam()
			if configDebug {
				buffer.WriteString(loggerGen(1, "entityAddRecord::Requests:"+XMLRequest))
			}
			XMLCreate, xmlmcErr := espXmlmc.Invoke("data", "entityAddRecord")
			if xmlmcErr != nil {
				buffer.WriteString(loggerGen(4, xmlmcErr.Error()))
				if configSplitLogs {
					uploadLogger(xmlmcErr.Error())
					uploadLogger(XMLRequest)
				}
				continue
			}
			var xmlRespon xmlmcRequestResponseStruct

			err := xml.Unmarshal([]byte(XMLCreate), &xmlRespon)
			if err != nil {
				mutexCounters.Lock()
				counters.createdSkipped++
				mutexCounters.Unlock()
				buffer.WriteString(loggerGen(4, err.Error()))
				if configSplitLogs {
					uploadLogger(err.Error())
					uploadLogger(XMLRequest)
				}
				continue
			}
			if xmlRespon.MethodResult != "ok" {
				mutexCounters.Lock()
				counters.createdSkipped++
				mutexCounters.Unlock()
				buffer.WriteString(loggerGen(4, "Log Request Failed ["+xmlRespon.State.ErrorRet+"]"))
				if configSplitLogs {
					uploadLogger(xmlRespon.State.ErrorRet)
					uploadLogger(XMLRequest)
				}
			} else {
				strNewCallRef = xmlRespon.RequestID
				buffer.WriteString(loggerGen(1, "Log Request Successful ["+strNewCallRef+"]"))
				mutexArrCallsLogged.Lock()
				arrCallsLogged[swCallID] = strNewCallRef
				mutexArrCallsLogged.Unlock()

				mutexCounters.Lock()
				counters.created++
				mutexCounters.Unlock()

				//Now update the request to create the activity stream
				espXmlmc.SetParam("socialObjectRef", "urn:sys:entity:"+appServiceManager+":Requests:"+strNewCallRef)
				espXmlmc.SetParam("content", "Request imported from Supportworks")
				espXmlmc.SetParam("visibility", "public")
				espXmlmc.SetParam("type", "Logged")
				if configDebug {
					buffer.WriteString(loggerGen(1, "activity::postMessage:"+espXmlmc.GetParam()))
				}
				fixed, err := espXmlmc.Invoke("activity", "postMessage")
				if err != nil {
					buffer.WriteString(loggerGen(5, "Activity Stream Creation failed for Request ["+strNewCallRef+"]"))
				} else {
					var xmlRespon xmlmcResponse
					err = xml.Unmarshal([]byte(fixed), &xmlRespon)
					if err != nil {
						buffer.WriteString(loggerGen(5, "Activity Stream Creation unmarshall failed for Request ["+strNewCallRef+"]"))
					} else {
						if xmlRespon.MethodResult != "ok" {
							buffer.WriteString(loggerGen(5, "Activity Stream Creation was unsuccessful for ["+strNewCallRef+"]: "+xmlRespon.MethodResult))
						} else {
							buffer.WriteString(loggerGen(1, "Activity Stream Creation successful"))
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
					if configDebug {
						buffer.WriteString(loggerGen(1, "entityUpdateRecord::Requests::logDate:"+espXmlmc.GetParam()))
					}
					XMLLogDate, xmlmcErr := espXmlmc.Invoke("data", "entityUpdateRecord")
					if xmlmcErr != nil {
						buffer.WriteString(loggerGen(4, "Unable to update Log Date of request ["+strNewCallRef+"] : "+xmlmcErr.Error()))
					}
					var xmlRespon xmlmcResponse

					errLogDate := xml.Unmarshal([]byte(XMLLogDate), &xmlRespon)
					if errLogDate != nil {
						buffer.WriteString(loggerGen(4, "Unable to update Log Date of request ["+strNewCallRef+"] : "+errLogDate.Error()))
					}
					if xmlRespon.MethodResult != "ok" {
						buffer.WriteString(loggerGen(4, "Unable to update Log Date of request ["+strNewCallRef+"] : "+xmlRespon.State.ErrorRet))
					}
				}

				//Now add status history
				addStatusHistory(strNewCallRef, strStatus, strLoggedDate, espXmlmc, &buffer)

				//Now do BPM Processing
				if strStatus != "status.resolved" &&
					strStatus != "status.closed" &&
					strStatus != "status.cancelled" {

					if strNewCallRef != "" && strServiceBPM != "" {
						espXmlmc.SetParam("application", appServiceManager)
						espXmlmc.SetParam("name", strServiceBPM)
						espXmlmc.SetParam("reference", strNewCallRef)
						espXmlmc.OpenElement("inputParam")
						espXmlmc.SetParam("name", "objectRefUrn")
						espXmlmc.SetParam("value", "urn:sys:entity:"+appServiceManager+":Requests:"+strNewCallRef)
						espXmlmc.CloseElement("inputParam")
						espXmlmc.OpenElement("inputParam")
						espXmlmc.SetParam("name", "requestId")
						espXmlmc.SetParam("value", strNewCallRef)
						espXmlmc.CloseElement("inputParam")
						XMLBPM, xmlmcErr := espXmlmc.Invoke("bpm", "processSpawn2")
						if xmlmcErr != nil {
							buffer.WriteString(loggerGen(4, "Unable to invoke BPM for request ["+strNewCallRef+"]: "+xmlmcErr.Error()))
						}
						var xmlRespon xmlmcBPMSpawnedStruct

						errBPM := xml.Unmarshal([]byte(XMLBPM), &xmlRespon)
						if errBPM != nil {
							buffer.WriteString(loggerGen(4, "Unable to read response when invoking BPM for request ["+strNewCallRef+"]:"+errBPM.Error()))
						}
						if xmlRespon.MethodResult != "ok" {
							buffer.WriteString(loggerGen(4, "Unable to invoke BPM for request ["+strNewCallRef+"]: "+xmlRespon.State.ErrorRet))
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
							if configDebug {
								buffer.WriteString(loggerGen(1, "entityUpdateRecord::Requests::bpmId:"+espXmlmc.GetParam()))
							}
							XMLBPMUpdate, xmlmcErr := espXmlmc.Invoke("data", "entityUpdateRecord")
							if xmlmcErr != nil {
								buffer.WriteString(loggerGen(4, "Unable to associated spawned BPM to request ["+strNewCallRef+"]: "+xmlmcErr.Error()))
							}
							var xmlRespon xmlmcResponse

							errBPMSpawn := xml.Unmarshal([]byte(XMLBPMUpdate), &xmlRespon)
							if errBPMSpawn != nil {
								buffer.WriteString(loggerGen(4, "Unable to read response from Hornbill instance when updating BPM on ["+strNewCallRef+"]:"+errBPMSpawn.Error()))
							}
							if xmlRespon.MethodResult != "ok" {
								buffer.WriteString(loggerGen(4, "Unable to associate BPM to Request ["+strNewCallRef+"]: "+xmlRespon.State.ErrorRet))
							}
						}
					}
				}

				// Now handle calls in an On Hold status
				if boolOnHoldRequest {
					espXmlmc.SetParam("requestId", strNewCallRef)
					espXmlmc.SetParam("onHoldUntil", strClosedDate)
					espXmlmc.SetParam("strReason", "Request imported from Supportworks in an On Hold status. See Historical Request Updates for further information.")
					if configDebug {
						buffer.WriteString(loggerGen(1, "OnHoldXMLMC: "+espXmlmc.GetParam()))
					}
					XMLBPM, xmlmcErr := espXmlmc.Invoke("apps/"+appServiceManager+"/Requests", "holdRequest")
					if xmlmcErr != nil {
						//log.Fatal(xmlmcErr)
						buffer.WriteString(loggerGen(4, "Unable to place request on hold ["+strNewCallRef+"] : "+xmlmcErr.Error()))
					}
					var xmlRespon xmlmcResponse

					errLogDate := xml.Unmarshal([]byte(XMLBPM), &xmlRespon)
					if errLogDate != nil {
						buffer.WriteString(loggerGen(4, "Unable to place request on hold ["+strNewCallRef+"] : "+errLogDate.Error()))
					}
					if xmlRespon.MethodResult != "ok" {
						buffer.WriteString(loggerGen(4, "Unable to place request on hold ["+strNewCallRef+"] : "+xmlRespon.State.ErrorRet))
					}
				}

				//Now apply historic updates
				request := RequestReferences{SwCallID: swCallID, SmCallID: strNewCallRef}
				applyHistoricalUpdates(request, espXmlmc, &buffer)
			}
		} else {
			//-- DEBUG XML TO LOG FILE
			var XMLSTRING = espXmlmc.GetParam()
			buffer.WriteString(loggerGen(1, "Request Log XML "+XMLSTRING))
			mutexCounters.Lock()
			counters.createdSkipped++
			mutexCounters.Unlock()
			espXmlmc.ClearParam()
		}
		bufferMutex.Lock()
		loggerWriteBuffer(buffer.String())
		bufferMutex.Unlock()
		buffer.Reset()
	}
}

func addStatusHistory(requestRef, requestStatus, dateLogged string, espXmlmc *apiLib.XmlmcInstStruct, buffer *bytes.Buffer) {
	espXmlmc.SetParam("application", "com.hornbill.servicemanager")
	espXmlmc.SetParam("entity", "RequestStatusHistory")
	espXmlmc.OpenElement("primaryEntityData")
	espXmlmc.OpenElement("record")
	espXmlmc.SetParam("h_request_id", requestRef)
	espXmlmc.SetParam("h_status", requestStatus)
	espXmlmc.SetParam("h_timestamp", dateLogged)
	espXmlmc.CloseElement("record")
	espXmlmc.CloseElement("primaryEntityData")
	XMLPub := espXmlmc.GetParam()
	XMLPublish, xmlmcErr := espXmlmc.Invoke("data", "entityAddRecord")
	if xmlmcErr != nil {
		buffer.WriteString(loggerGen(4, "XMLMC error: Unable to add status history record for ["+requestRef+"] : "+xmlmcErr.Error()))
		buffer.WriteString(loggerGen(1, XMLPub))
		return
	}
	var xmlRespon xmlmcResponse
	errLogDate := xml.Unmarshal([]byte(XMLPublish), &xmlRespon)
	if errLogDate != nil {
		buffer.WriteString(loggerGen(4, "Unmarshal error: Unable to add status history record for ["+requestRef+"] : "+errLogDate.Error()))
		buffer.WriteString(loggerGen(1, XMLPub))
		return
	}
	if xmlRespon.MethodResult != "ok" {
		buffer.WriteString(loggerGen(4, "MethodResult not OK: Unable to add status history record for ["+requestRef+"] : "+xmlRespon.State.ErrorRet))
		buffer.WriteString(loggerGen(1, XMLPub))
		return
	}
	buffer.WriteString(loggerGen(1, "Request Status History record success: ["+requestRef+"]"))
}
