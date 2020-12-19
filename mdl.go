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
	SetSQLQueriesFromTags(u interface{}) error
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
}

func (m *Mdl) IsInitialized() bool {
	return m.initialized
}

func (m *Mdl) SetInitialized(b bool) {
	m.initialized = b
}

func (m *Mdl) SetSQLQueriesFromTags(u interface{}) error {
	v := reflect.ValueOf(u)
	i := reflect.Indirect(v)
	s := i.Type()
	usName := m.getUnderscoredName(s.Name())
	usPluName := m.getPluralName(usName)

	m.dbTbl = usPluName
	m.dbColPrefix = usName
	m.url = usPluName

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
			log.Print(req)
			log.Print(lenmin)
			log.Print(lenmax)
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

func SaveMdlToDB(mdl IMdl, conn *sql.DB) error {
	if !mdl.IsInitialized() {
		err := mdl.SetSQLQueriesFromTags(mdl)
		if err != nil {
			return fmt.Errorf("error with SetSQLQueriesFromTags in SaveMdlToDB: %s", err)
		}
		mdl.SetInitialized(true)
	}
	return nil
}

/*import (
	"fmt"
	"log"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// ParseF0xTagLine parses f0x tag


// ValidateMdl loops through fields and validates them
func ValidateMdl(m interface{}) error {
	v := reflect.ValueOf(m)
	i := reflect.Indirect(v)
	s := i.Type()
	for j := 0; j < s.NumField(); j++ {
		field := s.Field(j)
		valueField := i.Field(j)
		if field.Name == "Mdl" || field.Name == "ID" || field.Name == "Flags" {
			continue
		}

		f0xTagLine := field.Tag.Get("f0x")
		req, _, lenmin, lenmax, err := ParseF0xTagLine(f0xTagLine)
		if err != nil {
			return fmt.Errorf("error with ParseF0xTagLine: %s", err)
		}

		if valueField.Kind() == reflect.String {
			if req && valueField.String() == "" {
				return fmt.Errorf("value req failed")
			}
			if lenmin > -1 && len(valueField.String()) < lenmin {
				return fmt.Errorf("value lenmin failed")
			}
			if lenmax > -1 && len(valueField.String()) > lenmax {
				return fmt.Errorf("value lenmax failed")
			}
		}
	}
	return nil
}

// SaveMdl updates the object in the database
func SaveMdl(m interface{}) error {
	err := ValidateMdl(m)
	if err != nil {
		return fmt.Errorf("validation failed: %s", err)
	}
	log.Print(m)
	return nil
}*/
