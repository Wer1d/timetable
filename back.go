package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/xuri/excelize/v2"
)

var groupIndex, totalIndex int
var mutex sync.Mutex
var courseData []map[string]interface{}
var majorData map[string][]interface{}
var f *excelize.File
var groupList [][]string // [[2390101 ชีววิทยา 34],[2390102 สัตววิทยา 21]]
var courseList [][]string
var fileName string
var record [][]string
var errorMsg string

func main() {

	lambda.Start(handleSend)

}

func findMajor(groupList [][]string) map[string][]interface{} {
	// groupList = [[2390101 ชีววิทยา 34],[2390102 สัตววิทยา 21]]
	data := make(map[string][]interface{})
	for i := 0; i < len(groupList); i++ { // access each send req of group
		//[ [course1],[course2] ] , [ [course3],[course4] ]
		groupListElem := groupList[i]
		for j := 0; j < len(groupListElem); j++ {
			if groupListElem[0] == "รวม" {
				break
			}
			groupCode := groupListElem[0]
			major := groupListElem[1]
			numStud := groupListElem[2]
			data[groupCode] = []interface{}{numStud, major}
		}
	}
	return data
}

func courseToDict() []map[string]interface{} { //majorData map[string][]interface{}
	data := []map[string]interface{}{}
	for i := 0; i < len(courseList); i++ {
		// [[1661 2301107 CALCULUS I 7 LEC MO WE FR 09:00-10:00 TAB-221 1 2 3 4],
		//  [1661 2302111 GEN CHEM I 3 LEC MO WE FR 11:00-12:00 MHMK-202 1 45 32 29 30 43 44 31 42]]

		rowData := map[string]interface{}{}
		row := courseList[i] // ["1661","230242","CALCULUS I","7","LEC","MO WE FR","09:00-10:00","TAB-221","20", "21", "22", "39","","",""]

		if len(row) > 6 && row[1] != "GEN ED" {

			rowData["Term"] = row[0]
			rowData["Course Number"] = row[1]
			rowData["Course Title"] = row[2]
			rowData["Section"] = row[3]
			rowData["Teach Type"] = row[4]
			rowData["Meeting Day"] = row[5]
			rowData["Meeting Time"] = row[6]
			rowData["Room"] = row[7]
			rowData["GROUP"] = strings.TrimSpace(strings.Join(row[8:], " ")) // tri, empty space after join

			data = append(data, rowData)
		}
	}
	return data
}

func handleSend(ctx context.Context, r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Println(r)
	log.Println(r.Body)
	decodedData, _ := base64.StdEncoding.DecodeString(r.Body)
	data := string(decodedData)

	log.Println("raw data : " + data)

	lines := strings.Split(data, "\r\n")
	log.Println("lines : ", lines)
	// Process each line
	modifiedLines := make([]string, 0, len(lines))
	for _, line := range lines {
		if line == "" || strings.HasPrefix(line, "----") || strings.HasPrefix(line, "Content") || strings.HasPrefix(line, "รวม") {
			continue
		}
		modifiedLines = append(modifiedLines, line)
	}
	log.Println("modifiedLines : ", modifiedLines) // delete header
	//extract header
	header := make([]string, 0, len(lines))
	for i, row := range modifiedLines {
		if i == 1 {
			break
		}
		header = strings.Split(row, ",")

	}
	log.Println("header : ", header)
	// find index of header
	for k, v := range header {
		if "GROUP" == v {
			groupIndex = k
		}
		if "Total" == v {
			totalIndex = k
		}
	}
	// raw data -> courseList, groupList -> majorData, courseData
	for _, row := range modifiedLines {
		if row == "" || strings.HasPrefix(row, "----") || strings.HasPrefix(row, "Content") || strings.HasPrefix(row, "Term") || strings.HasPrefix(row, "รวม") {
			continue
		}

		fields := strings.Split(row, ",") // arr ของแถว
		if fields[0] == "" {
			continue
		}
		log.Println("fields : ", fields, "\n len(fields) : ", len(fields)) // each row with empty string
		log.Println("groupIndex : ", groupIndex)
		log.Println("totalIndex : ", totalIndex)
		courseFields := make([]string, 0, len(fields)-5)
		groupFields := make([]string, 0, 3)
		for i, field := range fields {
			if i < groupIndex {
				courseFields = append(courseFields, field)
			} else if i < totalIndex { // จัดการข้อมูลก่อน group
				if field != "" {
					courseFields = append(courseFields, field)
				}
			} else if i > totalIndex {
				if field == "" || field == "รวม" {
					break
				}
				groupFields = append(groupFields, field)
			}

			// if len(fields) < 7 {
			// 	courseFields := make([]string, 0, len(fields)-5)
			// 	groupFields := make([]string, 0, 3)
			// 	if fields[len(fields)-2] == "รวม" {

			// 		for i, field := range fields {
			// 			log.Println(i, ":"+field)
			// 			if i < len(fields)-3 {
			// 				field = strings.TrimSpace(field)
			// 				if field != "" {
			// 					courseFields = append(courseFields, field)
			// 				}
			// 			}
			// 		}

			// 	} else if strings.HasPrefix(fields[len(fields)-3], "239") { // case ที่มี group data
			// 		log.Println("else if case passed")
			// 		for i, field := range fields {

			// 			if i >= len(fields)-3 {
			// 				log.Println("i : ", i)
			// 				log.Println("i >len(fields)-4", i > len(fields)-4)
			// 				groupFields = append(groupFields, field)
			// 				log.Println("groupFields : ", groupFields)

			// 			} else {
			// 				if i == len(fields)-4 {
			// 					continue
			// 				}
			// 				field = strings.TrimSpace(field)
			// 				if field != "" {
			// 					courseFields = append(courseFields, field)
			// 				}
			// 			}
			// 		}
			// 	} else { // case ที่มีแต่ course data
			// 		for i, field := range fields {
			// 			if i == len(fields)-1 {
			// 				break
			// 			}
			// 			field = strings.TrimSpace(field)
			// 			if field != "" {
			// 				courseFields = append(courseFields, field)
			// 			}
			// 		}
			// 	}
		}
		courseList = append(courseList, courseFields)
		groupList = append(groupList, groupFields)

		var newGroupList [][]string
		for _, elem := range groupList {
			if len(elem) > 0 {
				newGroupList = append(newGroupList, elem)
			}
		}
		groupList = newGroupList
	}

	// }
	log.Println("courseList : ", courseList)
	log.Println("groupList : ", groupList)
	courseData = courseToDict()
	majorData = findMajor(groupList) // findMajor() re-assign ค่าข้างใน เลยได้ majorData ก้อนใหม่

	// append sheet
	keys := make([]string, 0, len(majorData))
	for key := range majorData {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	log.Println("keys : ", keys)
	f = excelize.NewFile()
	for index, gid := range keys {
		f.NewSheet(gid) // add sheet ต่อจาก previous request
		f.SetSheetName(strconv.Itoa(index+1), gid)

	}
	fmt.Println(courseData)

	// gen table from each major(gid)
	for gid, _ := range majorData {
		fmt.Println("gid :", gid)

		entryList := []map[string]interface{}{} // [{},{}]
		// gid อยู่ใน row ในบ้าง
		for i := 0; i < len(courseData); i++ {
			c := courseData[i]
			groups := c["GROUP"]

			num, err := strconv.Atoi(string(gid[5:])) // 01,02,03,..
			// if gid ends with 0x
			if gid[5] == 48 {
				newGid := gid[6]
				num, err = strconv.Atoi(string(newGid))
				if err != nil {
					// Handle the error if the conversion fails
					fmt.Println("Error converting ASCII to int:", err)
				}
			}

			words := strings.Fields(groups.(string))
			set := make(map[string]bool)
			for _, str := range words {
				set[str] = true
			}
			if set[strconv.Itoa(num)] {
				entryList = append(entryList, c)
			}
		}
		// gen table
		if len(entryList) > 0 {
			PrintSchedule(entryList, gid, majorData[gid]) //   1 , 2 , 3 , ...
		}
	}
	// clear data for next request
	groupList = [][]string{}
	courseList = [][]string{}
	// if no error return file
	if errorMsg == "" {
		// Assuming f is the *excelize.File object
		responseData, _ := f.WriteToBuffer()

		// Encode the Excel data to base64 to be returned in the response body
		base64Data := base64.StdEncoding.EncodeToString(responseData.Bytes())

		return events.APIGatewayProxyResponse{
			StatusCode:      200,
			Body:            base64Data,
			IsBase64Encoded: true,
			Headers: map[string]string{
				"Content-Type":                  "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
				"Content-Disposition":           "attachment; filename=\"your_file_name.xlsx\"",
				"Filename":                      fileName,
				"Access-Control-Allow-Origin":   "*",
				"Access-Control-Expose-Headers": "Filename",
			},
		}, nil

	}
	// if error return error message
	errorMsgg := errorMsg
	errorMsg = ""

	return events.APIGatewayProxyResponse{
		StatusCode: 500,
		Body:       `{"error": "` + errorMsgg + `"}`,
	}, nil

}

func PrintSchedule(entryList []map[string]interface{}, gid string, majorName []interface{}) {
	mutex.Lock()
	defer mutex.Unlock()
	//    รอยแรกทำของตัวที่มี group เป็น gid ตัวท้าย
	fmt.Printf("Group %s Total subjects %d\n", gid, len(entryList))
	// TypeOf(entryList) = []map[string]interface{}

	term := fmt.Sprintf("%s", entryList[0]["Term"]) //string
	year := term[1:3]
	fileName = fmt.Sprintf("FirstYear-%s-%s.xlsx", year, term[3:])
	var semester string
	switch term[3:] {
	case "1":
		semester = "ต้น"
	case "2":
		semester = "ปลาย"
	case "3":
		semester = "ฤดูร้อน"
	}
	sheetName := gid
	// make header
	f.MergeCell(sheetName, "A1", "K1")
	f.SetCellValue(sheetName, "A1", "รหัสกลุ่มรายวิชา "+gid+" แบบที่ "+gid[5:]+" สาขาวิชา"+majorName[1].(string))
	f.MergeCell(sheetName, "A2", "K2")
	f.SetCellValue(sheetName, "A2", "ตารางสอนนิสิตชั้นปีที่ 1 คณะวิทยาศาสตร์ ภาคการศึกษา"+semester+" ปีการศึกษา 25"+year)

	f.SetCellValue(sheetName, "A4", "วัน-เวลา")
	// make time column
	for c := 1; c < 11; c++ {
		time := fmt.Sprintf("%d:00-%d:00", c+7, c+8)
		colLetter := string(rune('A' + c))
		cell := fmt.Sprintf("%s%d", colLetter, 4)
		f.SetCellValue(sheetName, cell, time)
	}
	// make day row
	days := []string{"จันทร์", "อังคาร", "พุธ", "พฤหัสบดี", "ศุกร์"}
	for i := 0; i < len(days); i++ {
		cell := fmt.Sprintf("%s%d", "A", 5+i)
		f.SetCellValue(sheetName, cell, days[i])
	}

	for i := 0; i < len(entryList); i++ {
		// loop row ที่มี group นั้น
		var entry = entryList[i]
		if entry["Meeting Day"].(string) == "" {
			errorMsg = fmt.Sprintf("Please insert Meeting Day for %s", entry["Course Number"].(string))
			return
		}
		var dayss = entry["Meeting Day"].(string)
		dayList := strings.Split(dayss, " ")

		for d := 0; d < len(dayList); d++ {

			decorateTable(entryList, gid, majorName, semester, year, sheetName)
			if entry["Meeting Time"].(string) == "" {
				errorMsg = fmt.Sprintf("Please insert Meeting time for %s", entry["Course Number"].(string))
				return
			}
			var timeSlot = getTime(entry["Meeting Time"].(string)) // 11-12
			var daySlot = decodeDay(dayList[d])                    // แปลงวันเป็นเลข เช่น MON = 0

			cell := fmt.Sprintf("%s%d", string(rune('A'+timeSlot[0]+1)), 5+daySlot) //

			var text = entry["Course Number"].(string)
			text += "\n" + entry["Course Title"].(string)
			if entry["Room"].(string) == "" {
				text += "\n" + "(Sec " + entry["Section"].(string) + ")" + "\n" + "AR"
			} else {

				text += "\n" + "(Sec " + entry["Section"].(string) + ")" + "\n" + entry["Room"].(string)
			}
			// check that cell is available
			if isCellAvailable(sheetName, cell, timeSlot[1]-timeSlot[0]) {

				if timeSlot[1]-timeSlot[0] > 1 { // have to merge

					startCell, _ := excelize.CoordinatesToCellName(timeSlot[0]+2, 5+daySlot)
					endCell, _ := excelize.CoordinatesToCellName(timeSlot[1]+1, 5+daySlot)

					f.MergeCell(sheetName, startCell, endCell)
					var text = entry["Course Number"].(string)

					style, _ := f.NewStyle(&excelize.Style{
						Fill: excelize.Fill{
							Type:    "pattern",
							Color:   []string{fmt.Sprintf("%s", decodeColor(entry["Course Number"].(string)))},
							Pattern: 1,
						},
						Border: []excelize.Border{
							{Type: "left", Color: "000000", Style: 2},
							{Type: "top", Color: "000000", Style: 2},
							{Type: "bottom", Color: "000000", Style: 2},
							{Type: "right", Color: "000000", Style: 2},
						},
						Font: &excelize.Font{
							Family: "Arial",
							Size:   8,
						},
						Alignment: &excelize.Alignment{
							Horizontal: "center",
							Vertical:   "center",
							WrapText:   true,
						},
					})
					f.SetCellStyle(sheetName, startCell, endCell, style)

					text += "\n" + entry["Course Title"].(string)
					if entry["Room"].(string) == "" {
						text += "\n" + "(Sec " + entry["Section"].(string) + ")" + "\n" + "AR"
					} else {

						text += "\n" + "(Sec " + entry["Section"].(string) + ")" + "\n" + entry["Room"].(string)
					}
					f.SetCellValue(sheetName, startCell, text)

				} else { // course is 1 hour
					var text = entry["Course Number"].(string)

					style, _ := f.NewStyle(&excelize.Style{
						Fill: excelize.Fill{Type: "pattern", Color: []string{fmt.Sprintf("%s", decodeColor(text))}, Pattern: 1},
						Border: []excelize.Border{
							{Type: "left", Color: "000000", Style: 2},
							{Type: "top", Color: "000000", Style: 2},
							{Type: "bottom", Color: "000000", Style: 2},
							{Type: "right", Color: "000000", Style: 2},
						},
						Font: &excelize.Font{
							Family: "Arial",
							Size:   8,
						},
						Alignment: &excelize.Alignment{
							Horizontal: "center",
							Vertical:   "center",
							WrapText:   true,
						},
					})
					f.SetCellStyle(sheetName, cell, cell, style)

					text += "\n" + entry["Course Title"].(string)
					if entry["Room"].(string) == "" {
						text += "\n" + "(Sec " + entry["Section"].(string) + ")" + "\n" + "AR"
					} else {

						text += "\n" + "(Sec " + entry["Section"].(string) + ")" + "\n" + entry["Room"].(string)
					}
					f.SetCellValue(sheetName, cell, text)
				}

				// Cell is not available, return an error message with conflicting values
			} else {
				oldCourse, _ := f.GetCellValue(sheetName, cell)
				fmt.Println("oldCourse ", oldCourse)
				if oldCourse == "" || oldCourse == entry["Course Number"].(string)+"\n"+entry["Course Title"].(string)+"\n"+"(Sec "+entry["Section"].(string)+")"+"\n"+entry["Room"].(string) {
					break
				}
				oldCourse = strings.ReplaceAll(oldCourse, "\n", "")
				errorMsg = fmt.Sprintf("Clash detected at GROUP %s for course %s %s (Sec %s) on %s %d-%d. Conflicting values: %s",
					gid[5:], entry["Course Number"].(string), entry["Course Title"].(string), entry["Section"].(string),
					dayList[d], timeSlot[0]+8, timeSlot[1]+8, oldCourse)
				return
			}
		}
	}

	// delete previous file and save new file
	_, err := os.Stat(fileName)
	if err == nil {
		err = os.Remove(fileName)
		if err != nil {
			fmt.Println("Failed to delete existing file:", err)
		}
	}
	err = f.SaveAs(fileName)
	if err != nil {
		fmt.Println("Failed to save workbook:", err)
	}

}

func isCellAvailable(sheetName, cell string, span int) bool {
	col, row, err := excelize.CellNameToCoordinates(cell)
	// 7 ,8
	if err != nil {
		return false
	}

	mergeCells, _ := f.GetMergeCells(sheetName)
	// check if cell is part of a merged region
	for _, mergeCell := range mergeCells {
		if mergeCell.GetStartAxis() == cell {
			return false // Cell is part of a merged region
		}
	}
	// check if adjacent cells have content or not
	for i := 0; i < span; i++ { //3
		rowName, _ := excelize.CoordinatesToCellName(col+i, row) //G8
		value, _ := f.GetCellValue(sheetName, rowName)
		if value != "" {
			return false // Cell has content or formatting applied
		}
	}
	return true
}
func decorateTable(entryList []map[string]interface{}, gid string, majorName []interface{}, semester string, year string, sheetName string) {

	// apply border for cell without border(no course)
	startRow := 5
	endRow := 9
	for row := startRow; row <= endRow; row++ {
		// Get the cell coordinates for each column in the row
		for col := 'A'; col <= 'K'; col++ {
			cell := fmt.Sprintf("%c%d", col, row)

			style, err := f.GetCellStyle(sheetName, cell)

			if err != nil {
				fmt.Println(err)
				return
			}

			// If the cell doesn't have a border style, apply it
			if style == 0 {
				style, _ := f.NewStyle(&excelize.Style{

					Border: []excelize.Border{
						{Type: "left", Color: "000000", Style: 2},
						{Type: "top", Color: "000000", Style: 2},
						{Type: "bottom", Color: "000000", Style: 2},
						{Type: "right", Color: "000000", Style: 2},
					},
					Alignment: &excelize.Alignment{
						Horizontal: "center",
						Vertical:   "center",
						WrapText:   true,
					},
				})
				f.SetCellStyle(sheetName, cell, cell, style)
			}
		}
	}

	// page layout
	var (
		size        = 9
		orientation = "landscape"
	)
	if err := f.SetPageLayout(sheetName, &excelize.PageLayoutOptions{
		Size:        &size,
		Orientation: &orientation,
	}); err != nil {
		fmt.Println(err)
	}
	f.SetCellValue(sheetName, "K9", "("+majorName[0].(string)+" คน)")

	// course List for fill description below table
	var courseList = make(map[string]interface{})
	for i := 0; i < len(entryList); i++ {
		entry := entryList[i]
		courseList[entry["Course Number"].(string)] = entry["Section"].(string)
	}

	var keys []string
	for course := range courseList {
		keys = append(keys, course)
	}
	sort.Strings(keys)

	// fill description below table
	i := 0
	courseCol := -4
	courseRow := 10
	maxRow := 5
	for _, course := range keys {
		sec := courseList[course]
		if i%maxRow == 0 {
			courseCol += 4
			f.SetCellValue(sheetName, fmt.Sprintf("%s%d", string(rune('C'+courseCol)), 10), "รหัสรายวิชา")
			f.SetCellValue(sheetName, fmt.Sprintf("%s%d", string(rune('D'+courseCol)), 10), "ตอนเรียน")
		}

		var r = courseRow + 1 + i%maxRow //
		f.SetCellValue(sheetName, fmt.Sprintf("%s%d", string(rune('B'+courseCol)), r), fmt.Sprintf("%d", i+1))
		f.SetCellValue(sheetName, fmt.Sprintf("%s%d", string(rune('C'+courseCol)), r), course)
		f.SetCellValue(sheetName, fmt.Sprintf("%s%d", string(rune('D'+courseCol)), r), sec)
		i += 1
	}

	// หมายเหตุ
	f.SetCellValue(sheetName, "A16", "หมายเหตุ")
	style, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Family:    "Arial",
			Size:      10,
			Bold:      true,
			Underline: "single",
		},
	})
	f.SetCellStyle(sheetName, "A16", "A16", style)

	// description below table
	f.MergeCell(sheetName, "B16", "L17")
	style, err = f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Family: "Arial",
			Size:   10,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "center",
			WrapText:   true,
		},
	})
	f.SetCellValue(sheetName, "B16", "ให้นิสิตตรวจสอบห้องเรียนบน reg.chula ช่วงเปิดภาคเรียนอีกครั้ง, หากรายวิชาขึ้น Course Full ให้นิสิตลงทะเบียน section อื่นที่วัน/เวลาเดียวกันในรอบที่ 2 (21-26 ก.ค.) ได้")
	f.SetCellStyle(sheetName, "B11", "L17", style)

	//titlestyle
	titleStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Family: "Arial",
			Size:   15,
			Bold:   true,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})
	f.SetCellStyle(sheetName, "A1", "H2", titleStyle)

	// time style
	timeStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Family: "Arial",
			Size:   11,
			Bold:   true,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 2},
			{Type: "top", Color: "000000", Style: 2},
			{Type: "bottom", Color: "000000", Style: 2},
			{Type: "right", Color: "000000", Style: 2},
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
			WrapText:   true,
		},
	})
	f.SetCellStyle(sheetName, "A4", "K4", timeStyle)

	// day style
	dayStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Family: "Arial",
			Size:   12,
			Bold:   true,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 2},
			{Type: "top", Color: "000000", Style: 2},
			{Type: "bottom", Color: "000000", Style: 2},
			{Type: "right", Color: "000000", Style: 2},
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
			// WrapText:   true,
		},
	})
	f.SetCellStyle(sheetName, "A5", "A9", dayStyle)

	// resize col
	f.SetColWidth(sheetName, "A", "K", 12)

	//resize row
	for i := 4; i < 10; i++ {
		f.SetRowHeight(sheetName, i, 52)
	}

	if err != nil {
		fmt.Println(err)
	}

}
func getTime(timeStr string) []int {
	timeArr := strings.Split(timeStr, "-")
	startTime := strings.Split(timeArr[0], ":")
	endTime := strings.Split(timeArr[1], ":")
	start, _ := strconv.Atoi(startTime[0])
	end, _ := strconv.Atoi(endTime[0])

	return []int{start - 8, end - 8}
}

func decodeDay(dayStr string) int {
	switch dayStr {
	case "MO":
		return 0
	case "TU":
		return 1
	case "WE":
		return 2
	case "TH":
		return 3
	case "FR":
		return 4
	default:
		return 0
	}

}
func decodeColor(courseID string) string {
	if courseID[0:4] == "2301" {
		return "EAD1DC"
	} else if courseID[0:4] == "2302" {
		return "FFF2CC"
	} else if courseID[0:4] == "2304" {
		return "C9DAF8"
	} else if courseID[0:4] == "2303" {
		return "D9EAD3"
	} else if courseID[0:4] == "2305" {
		return "D9EAD3"
	} else if courseID[0:4] == "5500" {
		return "EFEFEF"
	} else {
		return "FFFFFF"
	}
}
