package main

import (
	"fmt"
	"reflect"
	"strings"
	"strconv"
	"regexp"
	"unicode"
)

type ModelHelper struct {
	queryDropTable   string
	queryCreateTable string
	queryInsert      string
	queryUpdateById  string
	querySelectById  string
	queryDeleteById  string
	dbTbl            string
	dbColPrefix      string
	url              string
	reqFields        []int
	lenFields        [][3]int
	emailFields      []int
	linkFields       [][2]int
	valFields        [][3]int
	regexpFields     map[int]*regexp.Regexp
}

func NewModelHelper(m interface{}) (*ModelHelper, error) {
	h := &ModelHelper{}
	err := h.SetFromTags(m)
	if err != nil {
		return nil, fmt.Errorf("error with SetFromTags in NewModelHelper: %s", err)
	}
	return h, nil
}

func (m *ModelHelper) GetQueryDropTable() string {
	return m.queryDropTable
}

func (m *ModelHelper) GetQueryCreateTable() string {
	return m.queryCreateTable
}

func (m *ModelHelper) GetQueryInsert() string {
	return m.queryInsert
}

func (m *ModelHelper) GetQueryUpdateById() string {
	return m.queryUpdateById
}

func (m *ModelHelper) GetQuerySelectById() string {
	return m.querySelectById
}

func (m *ModelHelper) GetQueryDeleteById() string {
	return m.queryDeleteById
}

func (m *ModelHelper) SetFromTags(u interface{}) error {
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
	m.linkFields = make([][2]int, 0)
	m.valFields = make([][3]int, 0)

	queryCreateTableCols := ""
	querySelectCols := ""
	queryUpdateCols := ""
	updateFieldCnt := 0
	queryInsertCols := ""
	queryInsertVals := ""
	insertFieldCnt := 0

	m.regexpFields = make(map[int]*regexp.Regexp, 0)

	for j := 0; j < s.NumField(); j++ {
		field := s.Field(j)
		if field.Type.Kind() != reflect.Int64 && field.Type.Kind() != reflect.String && field.Type.Kind() != reflect.Int {
			continue
		}

		crudlTagLine := field.Tag.Get("crudl")
		req, lenmin, lenmax, link, email, valmin, valmax, re, err := m.parseCrudlTagLine(crudlTagLine)
		if err != nil {
			return fmt.Errorf("error with parseCrudlTagLine: %s", err)
		}
		if req {
			m.reqFields = append(m.reqFields, j)
		}
		if email {
			m.emailFields = append(m.emailFields, j)
		}
		if lenmin > -1 || lenmax > -1 {
			m.lenFields = append(m.lenFields, [3]int{j, lenmin, lenmax})
		}
		if valmin > -1 || valmax > -1 {
			m.valFields = append(m.valFields, [3]int{j, valmin, valmax})
		}
		if link != "" {
			linkedField, linkedFound := s.FieldByName(link)
			if !linkedFound {
				return fmt.Errorf("invalid link %s", link)
			}
			m.linkFields = append(m.linkFields, [2]int{j, linkedField.Index[0]})
		}
		if re != "" {
			m.regexpFields[j] = regexp.MustCompile(re)
		}

		dbCol := ""
		if field.Name == "ID" {
			dbCol = m.dbColPrefix + "_id"
		} else if field.Name == "Flags" {
			dbCol = m.dbColPrefix + "_flags"
		} else {
			dbCol = m.getUnderscoredName(field.Name)
		}

		dbColParams := ""
		if field.Name == "ID" {
			dbColParams = "SERIAL PRIMARY KEY"
		} else if field.Name == "Flags" {
			dbColParams = "BIGINT"
		} else {
			switch field.Type.String() {
			case "string":
				dbColParams = "VARCHAR(255)"
			case "int64":
				dbColParams = "BIGINT"
			case "int":
				dbColParams = "BIGINT"
			default:
				dbColParams = "VARCHAR(255)"
			}
		}

		if queryCreateTableCols != "" {
			queryCreateTableCols += ", "
		}
		queryCreateTableCols += dbCol + " " + dbColParams
		if querySelectCols != "" {
			querySelectCols += ", "
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

	m.queryDropTable = "DROP TABLE IF EXISTS " + m.dbTbl
	m.queryCreateTable = "CREATE TABLE " + m.dbTbl + " (" + queryCreateTableCols + ")"
	m.queryDeleteById = "DELETE FROM " + m.dbTbl + " WHERE " + m.dbColPrefix + "_id = $1"
	m.querySelectById = "SELECT " + querySelectCols + " FROM " + m.dbTbl + " WHERE " + m.dbColPrefix + "_id = $1"
	m.queryInsert = "INSERT INTO " + m.dbTbl + "(" + queryInsertCols + ") VALUES (" + queryInsertVals + ") RETURNING " + m.dbColPrefix + "_id"
	updateFieldCnt++
	m.queryUpdateById = "UPDATE " + m.dbTbl + " SET " + queryUpdateCols + " WHERE " + m.dbColPrefix + "_id = $" + strconv.Itoa(updateFieldCnt)
	return nil
}

func (m *ModelHelper) getUnderscoredName(s string) string {
	o := ""
	var prev rune
	for i, ch := range s {
		if i == 0 {
			o += strings.ToLower(string(ch))
		} else {
			if unicode.IsUpper(ch) {
				if prev == 'I' && ch == 'D' {
					o += strings.ToLower(string(ch))
				} else {
					o += "_" + strings.ToLower(string(ch))
				}
			} else {
				o += string(ch)
			}
		}
		prev = ch
	}
	return o
}

func (m *ModelHelper) getPluralName(s string) string {
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

func (m *ModelHelper) parseCrudlTagLine(s string) (bool, int, int, string, bool, int, int, string, error) {
	xt := strings.SplitN(s, " ", -1)
	req := false
	lenmin := -1
	lenmax := -1
	valmin := -1
	valmax := -1
	re := ""
	link := ""
	email := false
	if len(xt) > 0 {
		for _, t := range xt {
			if t == "req" {
				req = true
			}
			if t == "email" {
				email = true
			}
			for _, sl := range []string{"lenmin", "lenmax", "valmin", "valmax", "regexp"} {
				if strings.HasPrefix(t, sl+":") {
					lStr := strings.Replace(t, sl+":", "", 1)
					/*matched, err := regexp.Match(`^[0-9]+$`, []byte(lStr))
					if err != nil {
						return false, 0, 0, "", false, 0, 0, "", fmt.Errorf("error with regexp.Match on " + sl)
					}
					if !matched {
						return false, 0, 0, "", false, 0, 0, "", fmt.Errorf(sl + " has invalid value")
					}*/
					if sl == "lenmin" {
						lenmin, _ = strconv.Atoi(lStr)
					} else if sl == "lenmax" {
						lenmax, _ = strconv.Atoi(lStr)
					} else if sl == "valmin" {
						valmin, _ = strconv.Atoi(lStr)
					} else if sl == "valmax" {
						valmax, _ = strconv.Atoi(lStr)
					} else if sl == "regexp" {
						re = lStr
					}
				}
			}
			if strings.HasPrefix(t, "link:") {
				lStr := strings.Replace(t, "link:", "", 1)
				matched, err := regexp.Match(`^[a-zA-Z0-9]+$`, []byte(lStr))
				if err != nil {
					return false, 0, 0, "", false, 0, 0, "", fmt.Errorf("error with regexp.Match on link")
				}
				if !matched {
					return false, 0, 0, "", false, 0, 0, "", fmt.Errorf("link has invalid value")
				}
				link = lStr
			}
		}
	}
	return req, lenmin, lenmax, link, email, valmin, valmax, re, nil
}
