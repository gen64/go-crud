package main

import (
	"database/sql"
	"log"
	"fmt"
	"reflect"
	"strings"
	"strconv"
	"regexp"
	"unicode"
)

type IMdl interface {
	IsInitialized() bool
	SetInitialized(b bool)
	SetFromTags(u interface{}) error
}

type Mdl struct {
	initialized bool
	queryInsert string
	queryUpdateById string
	querySelectById string
	queryDeleteById string
	dbTbl string
	dbColPrefix string
	url string
	reqFields []int
	lenFields [][3]int
}

func (m *Mdl) IsInitialized() bool {
	return m.initialized
}

func (m *Mdl) SetInitialized(b bool) {
	m.initialized = b
}

func (m *Mdl) SetFromTags(u interface{}) error {
	v := reflect.ValueOf(u)
	i := reflect.Indirect(v)
	s := i.Type()
	usName := m.getUnderscoredName(s.Name())
	usPluName := m.getPluralName(usName)

	m.dbTbl = usPluName
	m.dbColPrefix = usName
	m.url = usPluName

	m.reqFields = make([]int, 0)
	m.lenFields = make([][3]int, 0)

	querySelectCols := ""
	queryUpdateCols := ""
	updateFieldCnt := 0
	queryInsertCols := ""
	queryInsertVals := ""
	insertFieldCnt := 0

	for j := 0; j < s.NumField(); j++ {
		field := s.Field(j)
		dbCol := ""
		if field.Name == "ID" {
			dbCol = m.dbColPrefix + "_id"
		} else if field.Name == "Flags" {
			dbCol = "flags"
		} else if field.Name != "Mdl" {
			dbCol = m.getUnderscoredName(field.Name)
		}

		if field.Name != "Mdl" {
			if querySelectCols != "" {
				querySelectCols += ","
			}
			querySelectCols += dbCol
			if field.Name != "ID" {
				updateFieldCnt++
				if queryUpdateCols != "" {
					queryUpdateCols += ","
				}
				queryUpdateCols += dbCol + "=$" + strconv.Itoa(updateFieldCnt)
				insertFieldCnt++
				if queryInsertCols != "" {
					queryInsertCols += ","
				}
				queryInsertCols += dbCol
				if queryInsertVals != "" {
					queryInsertVals += ","
				}
				queryInsertVals += "$" + strconv.Itoa(insertFieldCnt)
			}
		}

		if field.Name != "Mdl" && field.Name != "ID" && field.Name != "Flags" {
			f0xTagLine := field.Tag.Get("f0x")
			req, lenmin, lenmax, err := m.parseF0xTagLine(f0xTagLine)
			if err != nil {
				return fmt.Errorf("error with parseF0xTagLine: %s", err)
			}
			if req {
				m.reqFields = append(m.reqFields, j)
			}
			if lenmin > -1 || lenmax > -1 {
				m.lenFields = append(m.lenFields, [3]int{j, lenmin, lenmax})
			}
		}
	}

	m.queryDeleteById = "DELETE FROM " + m.dbTbl + " WHERE " + m.dbColPrefix + "_id = $1"
	m.querySelectById = "SELECT " + querySelectCols + " FROM " + m.dbTbl + " WHERE " + m.dbColPrefix + "_id = $1"
	m.queryInsert = "INSERT INTO " + m.dbTbl + "(" + queryInsertCols + ") VALUES (" + queryInsertVals + ")"
	updateFieldCnt++
	m.queryUpdateById = "UPDATE " + m.dbTbl + " SET " + queryUpdateCols + " WHERE " + m.dbColPrefix + "_id = $" + strconv.Itoa(updateFieldCnt)
	log.Print(m)
	return nil
}

func (m *Mdl) getUnderscoredName(s string) string {
	o := ""
	for i, ch := range s {
		if i == 0 {
			o += strings.ToLower(string(ch))
		} else {
			if unicode.IsUpper(ch) {
				o += "_" + strings.ToLower(string(ch))
			} else {
				o += string(ch)
			}
		}
	}
	return o
}

func (m *Mdl) getPluralName(s string) string {
	re := regexp.MustCompile(`y$`)
	if re.MatchString(s) {
		return string(re.ReplaceAll([]byte(s), []byte(`ies`)))
	}
	re = regexp.MustCompile(`s$`)
	if re.MatchString(s) {
		return s + "es"
	}
	return s + "s"
}

func (m *Mdl) parseF0xTagLine(s string) (bool, int, int, error) {
	xt := strings.Split(s, " ")
	req := false
	lenmin := -1
	lenmax := -1
	if len(xt) > 0 {
		for _, t := range xt {
			if t == "req" {
				req = true
			}
			for _, sl := range []string{"lenmin", "lenmax"} {
				if strings.HasPrefix(t, sl+":") {
					lStr := strings.Replace(t, sl+":", "", 1)
					matched, err := regexp.Match(`^[0-9]+$`, []byte(lStr))
					if err != nil {
						return false, 0, 0, fmt.Errorf("error with regexp.Match on " + sl)
					}
					if !matched {
						return false, 0, 0, fmt.Errorf(sl + " has invalid value")
					}
					if sl == "lenmin" {
						lenmin, _ = strconv.Atoi(lStr)
					} else if sl == "lenmax" {
						lenmax, _ = strconv.Atoi(lStr)
					}
				}
			}
		}
	}
	return req, lenmin, lenmax, nil
}

func ValidateMdl(mdl IMdl) error {
	return nil
}

func SaveMdlToDB(mdl IMdl, conn *sql.DB) error {
	if !mdl.IsInitialized() {
		err := mdl.SetFromTags(mdl)
		if err != nil {
			return fmt.Errorf("error with SetSQLQueriesFromTags in SaveMdlToDB: %s", err)
		}
		mdl.SetInitialized(true)
	}
	return nil
}

func SetMdlFromDB(mdl IMdl, conn *sql.DB, id string) error {

}

func DeleteMdlFromDB(mdl IMdl, conn *sql.DB) error {

}
