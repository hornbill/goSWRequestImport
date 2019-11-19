package main

import (
	"bytes"
	"encoding/xml"
	"fmt"

	apiLib "github.com/hornbill/goApiLib"
)

// searchTeam -- Function to check if passed-through support team name is on the instance
func searchGroup(groupID string, espXmlmc *apiLib.XmlmcInstStruct, buffer *bytes.Buffer) (groupFound bool, groupName string) {
	groupFound = false
	groupName = ""

	//Check Cache
	companyIsInCache, GroupNameCache := recordInCache(groupID, "Company")
	//-- Check if we have cached the Team already
	if companyIsInCache {
		groupName = GroupNameCache
		groupFound = true
		return
	}

	//-- ESP Query for team
	espXmlmc.SetParam("id", groupID)
	XMLTeamSearch, xmlmcErr := espXmlmc.Invoke("admin", "groupGetInfo")
	if xmlmcErr != nil {
		buffer.WriteString(loggerGen(4, "Unable to Search for Group: "+fmt.Sprintf("%v", xmlmcErr)))
		return groupFound, groupName
	}
	var xmlRespon xmlmcGroupListResponse

	err := xml.Unmarshal([]byte(XMLTeamSearch), &xmlRespon)
	if err != nil {
		buffer.WriteString(loggerGen(4, "Unable to Search for Group: "+fmt.Sprintf("%v", err)))
	} else {
		if xmlRespon.MethodResult != "ok" {
			buffer.WriteString(loggerGen(5, "Unable to Search for Group: "+xmlRespon.State.ErrorRet))
		} else {
			//-- Check Response
			if xmlRespon.Name != "" {
				groupFound = true
				//-- Add Team to Cache
				groupName = xmlRespon.Name
				var newGroupForCache groupListStruct
				newGroupForCache.ID = groupID
				newGroupForCache.Name = groupName
				teamNamedMap := []groupListStruct{newGroupForCache}
				mutexTeams.Lock()
				teams = append(teams, teamNamedMap...)
				mutexTeams.Unlock()
			}
		}
	}
	return groupFound, groupName
}
