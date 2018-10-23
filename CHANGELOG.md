## 1.5.1 (October 23rd 2018)

Defect fix:
- Fixed issue with missing log entries in dryrun mode


## 1.5.0 (October 14th 2018)

Features:
- Performance improvements
- Memory usage improvements
- Reduction in the number of HTTP sessions used
- No longer required to provide instance zone when running tool
- Logging improvements
- Historic Updates and File Attachments are now processed for each request rather than at the end, improving log readability

Defect fix:
- Fixed memory leak

## 1.4.7 (February 7th 2018)

Defect fix:
- Fixed issue with call diary entries sometimes assigning to incorrect request

## 1.4.6 (February 6th 2018)

Features:
- Further improved memory usage

## 1.4.5 (February 2nd 2018)

Features:
- Improved multithreading peformance and memory usage

## 1.4.4 (December 12th 2017)

Defect fix:
- Fixed issue where team mapping could return other groups of the same name.

## 1.4.3 (December 8th 2017)

Features:
- Now removes any incompatible (non UTF-8) characters from the decoded Supportworks Mail file response
- Improved output of SWM file text representation, included RFC headers and made line feeds Windows compatible.

Defect fixes:
- Fixed "null" value issue for date/time of file attachments
- Fixed warnings thrown when date/time is string instead of EPOCH

## 1.4.2 (December 7th 2017)

Defect Fix:
- Corrected issue with historic update attachments being associated to the incorrect historic update within the same request

Feature:
- Improved output when unable to connect to the Hornbill instance

## 1.4.1 (November 30th 2017)

Defect Fix:
- Now sending UTC time to Hornbill APIs to ensure correct log date being applied

## 1.4.0 (November 28th 2017)

Defect Fix:
- Fixed mismatch of historic attachment UpdateID to Historic Update Index

Features:
- Refactored code, broken the code up in the original release in to modules for easier reading and maintenance;
- Status Mapping, which was hard-coded, is now configurable;
- Output DB connection errors to CLI as well as Log
- Optimise hornbillitsmhistoric index once historic updates have been added
- Made the Request Type in the configuration an array of objects, so can add configuration for as many request type imports as necessary.
- Added support for import of Releases;
- Added support for import of BP Task;
- Added support for admin-defined query to return request associations, and configuration to allow imported BP Tasks to be associaed to other imported request types;
- Process file attachments in-line when importing requests, rather than at the end of the import process;
- Supportworks Mail file extraction, and attach files contained in SWM emails as files against new request or historic updates
- Improved logging

## 1.3.0 (January 3rd 2017)

Feature:
- Added logic to write AdditionalFieldMapping mapped data to new RequestsExtended entity

## 1.2.10 (November 25th 2016)

Defect Fix:
- Issue importing requests in an On Hold status in to Service Manager v2.30+

## 1.2.9 (November 14th 2016)

Defect Fix:
- Issue importing file attachments in to historical attachments entity

## 1.2.8 (November 7th 2016)

Feature:
- Historical diary entries are now ordered by descending update time

## 1.2.7 (July 7th, 2016)

Defect Fix:
- File attachments with names containing API-constrained characters [<>|\/:*?"] were not imported. These characters are now replaced by an underscore character _

## 1.2.6 (June 17th, 2016)

Feature:
- Added -attachments flag to allow for the importing of attachments without prompt

Defect Fix:
- Certain imported requests did not have an active activity stream, meaning they could not be updated.

## 1.2.5 (May 31st, 2016)

Defect Fixes:
- Writing of duplicate Request profile codes failing
- Historic Update File Attachments with a .SWM extension importing but not accessible through Service Manager Request UI

## 1.2.4 (May 25th, 2016)

Features:
- Release binary now includes 32bit and 64bit Windows executables
- Added -concurrent flag, allowing you to specify the maximum number of requests to be imported concurrently
- Improved performance when importing file attachments
- Added ability to specify Service Manager Profile Code separation character
- Improved client-side import logging.

## 1.2.3 (May 9th, 2016)

Defect Fixes:
- Fixed data mapping issues
- Fixed issue where imported calls in a status of On Hold could not be taken off hold  

## 1.2.2 (April 29th, 2016)

Features:
- Allow the back-dating of imported requests,to the date/time the original Supportworks request was logged
- Allow the import of Resolved Date & Closed Date to match those of the original Supportworks request
- New requests logged from requests that are On Hold in Supportworks are now placed On Hold in Service Manager, to the original requests date & time

Defect Fixes:
- Fixed issue when importing historical diary entries that have a Time Spent value of NULL

## 1.2.1 (April 28th, 2016)

Features:
- Takes request prefix from Application Settings rather than import tool

## 1.2.0 (April 27th, 2016)

Features:
- Improved import performance:
    - Streamlined API's and client side record caching
    - Multi-threaded the Request Association and File Attachment Import code
- Extended field mapping, allowing more request fields to be written to, including class-specific extended table fields

Defect Fixes:
- Fixed race conditions in Goroutines
- Fixed issue with MSSQL driver returning INT64 values, causing data conversion problems

## 1.1.2 (April 7th, 2016)

Defect Fix:
- Corrected output of oldCallRef mapping variable

## 1.1.1 (April 7th, 2016)

Features:
- Added additional stage in file attachment import user confirmation;
- Enhanced notification display using color in CLI output

## 1.1.0 (April 6th, 2016)

Features:
- Improved import performance using Goroutines & parallel processing of request logging

## 1.0.2 (February 24th, 2016)

Defect Fixes:
- No Default Service being assigned to imported requests when Service Name from Supportworks data contained a NULL value
- Updated Request Status Matrix.

## 1.0.1 (February 1st, 2016)

Features:
- Added missing brace to ConfKnownError section of configuration file.

## 1.0.0 (January 22, 2016)

Features:
- Initial Release
