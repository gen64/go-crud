package crudl

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// HelperError has details on failure in reflecting the struct
type HelperError struct {
	Op  string
	Tag string
	err error
}

func (e HelperError) Error() string {
	return e.err.Error()
}

func (e HelperError) Unwrap() error {
	return e.err
}

// Helper is used to generate Postgres SQL queries and parse validation defined in "crudl" tag so that can be attached to a specific struct type
type Helper struct {
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
	err              *HelperError
}

// NewHelper returns new Helper struct
func NewHelper(m interface{}, p string) *Helper {
	h := &Helper{}
	h.reflectStruct(m, p)
	return h
}

// Err returns error that occurred when reflecting struct
func (m *Helper) Err() *HelperError {
	return m.err
}

// GetQueryDropTable returns drop table query
func (m *Helper) GetQueryDropTable() string {
	return m.queryDropTable
}

// GetQueryCreateTable return create table query
func (m *Helper) GetQueryCreateTable() string {
	return m.queryCreateTable
}

// GetQueryInsert returns insert query
func (m *Helper) GetQueryInsert() string {
	return m.queryInsert
}

// GetQueryUpdateById returns update query
func (m *Helper) GetQueryUpdateById() string {
	return m.queryUpdateById
}

// GetQuerySelectById returns select query
func (m *Helper) GetQuerySelectById() string {
	return m.querySelectById
}

// GetQueryDeleteById returns delete query
func (m *Helper) GetQueryDeleteById() string {
	return m.queryDeleteById
}

func (m *Helper) setFieldFromTag(s string, j int) string {
	req, lenmin, lenmax, link, email, valmin, valmax, re, err := m.parseTag(s)
	if err != nil {
		m.err = err
		return ""
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
	if re != "" {
		m.regexpFields[j] = regexp.MustCompile(re)
	}
	return link
}

func (m *Helper) getDBCol(n string) string {
	dbCol := ""
	if n == "ID" {
		dbCol = m.dbColPrefix + "_id"
	} else if n == "Flags" {
		dbCol = m.dbColPrefix + "_flags"
	} else {
		dbCol = m.getUnderscoredName(n)
	}
	return dbCol
}

func (m *Helper) getDBColParams(n string, t string) string {
	dbColParams := ""
	if n == "ID" {
		dbColParams = "SERIAL PRIMARY KEY"
	} else if n == "Flags" {
		dbColParams = "BIGINT"
	} else {
		switch t {
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
	return dbColParams
}

func (m *Helper) addWithComma(s string, v string) string {
	if s != "" {
		s += ","
	}
	s += v
	return s
}

func (m *Helper) reflectStruct(u interface{}, dbTablePrefix string) {
	v := reflect.ValueOf(u)
	i := reflect.Indirect(v)
	s := i.Type()

	usName := m.getUnderscoredName(s.Name())
	usPluName := m.getPluralName(usName)
	m.dbTbl = dbTablePrefix + usPluName
	m.dbColPrefix = usName
	m.url = usPluName

	m.reqFields = make([]int, 0)
	m.lenFields = make([][3]int, 0)
	m.linkFields = make([][2]int, 0)
	m.valFields = make([][3]int, 0)

	var queryCreateTableCols, querySelectCols, queryUpdateCols, queryInsertCols, queryInsertVals string
	var updateFieldCnt, insertFieldCnt int

	m.regexpFields = make(map[int]*regexp.Regexp, 0)

	for j := 0; j < s.NumField(); j++ {
		field := s.Field(j)
		if field.Type.Kind() != reflect.Int64 && field.Type.Kind() != reflect.String && field.Type.Kind() != reflect.Int {
			continue
		}

		link := m.setFieldFromTag(field.Tag.Get("crudl"), j)
		if m.err != nil {
			return
		}

		if link != "" {
			linkedField, linkedFound := s.FieldByName(link)
			if !linkedFound {
				m.err = &HelperError{
					Op:  "GetLink",
					Tag: link,
					err: errors.New("invalid link value"),
				}
				return
			}
			m.linkFields = append(m.linkFields, [2]int{j, linkedField.Index[0]})
		}

		dbCol := m.getDBCol(field.Name)
		dbColParams := m.getDBColParams(field.Name, field.Type.String())

		queryCreateTableCols = m.addWithComma(queryCreateTableCols, dbCol+" "+dbColParams)
		querySelectCols = m.addWithComma(querySelectCols, dbCol)
		if field.Name != "ID" {
			updateFieldCnt++
			queryUpdateCols = m.addWithComma(queryUpdateCols, dbCol+"=$"+strconv.Itoa(updateFieldCnt))

			insertFieldCnt++
			queryInsertCols = m.addWithComma(queryInsertCols, dbCol)
			queryInsertVals = m.addWithComma(queryInsertVals, "$"+strconv.Itoa(insertFieldCnt))
		}
	}

	m.queryDropTable = "DROP TABLE IF EXISTS " + m.dbTbl
	m.queryCreateTable = "CREATE TABLE " + m.dbTbl + " (" + queryCreateTableCols + ")"
	m.queryDeleteById = "DELETE FROM " + m.dbTbl + " WHERE " + m.dbColPrefix + "_id = $1"
	m.querySelectById = "SELECT " + querySelectCols + " FROM " + m.dbTbl + " WHERE " + m.dbColPrefix + "_id = $1"
	m.queryInsert = "INSERT INTO " + m.dbTbl + "(" + queryInsertCols + ") VALUES (" + queryInsertVals + ") RETURNING " + m.dbColPrefix + "_id"
	m.queryUpdateById = "UPDATE " + m.dbTbl + " SET " + queryUpdateCols + " WHERE " + m.dbColPrefix + "_id = $" + strconv.Itoa(updateFieldCnt+1)
}

func (m *Helper) getUnderscoredName(s string) string {
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

func (m *Helper) getPluralName(s string) string {
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

func (m *Helper) parseTag(s string) (bool, int, int, string, bool, int, int, string, *HelperError) {
	xt := strings.SplitN(s, " ", -1)
	xb := map[string]bool{
		"req":   false,
		"email": false,
	}
	xi := map[string]int{
		"lenmin": -1,
		"lenmax": -1,
		"valmin": -1,
		"valmax": -1,
	}
	xs := map[string]string{
		"regexp": "",
		"link":   "",
	}
	var helperError *HelperError

	if len(xt) < 1 {
		return xb["req"], xi["lenmin"], xi["lenmax"], xs["link"], xb["email"], xi["valmin"], xi["valmax"], xs["regexp"], helperError
	}

	for _, t := range xt {
		if helperError != nil {
			break
		}

		if t == "req" || t == "email" {
			xb[t] = true
		}

		for _, sl := range []string{"lenmin", "lenmax", "valmin", "valmax", "regexp", "link:"} {
			if helperError != nil {
				break
			}
			if strings.HasPrefix(t, sl+":") {
				lStr := strings.Replace(t, sl+":", "", 1)
				if sl == "regexp" {
					xs["regexp"] = lStr
				} else if sl == "link" {
					matched, err := regexp.Match(`^[a-zA-Z0-9]+$`, []byte(lStr))
					if err != nil {
						helperError = &HelperError{
							Op:  "ParseTag",
							Tag: "link",
							err: fmt.Errorf("regexp.Match failed: %w", err),
						}
					}
					if !matched {
						helperError = &HelperError{
							Op:  "ParseTag",
							Tag: "link",
							err: errors.New("not int and gt 0"),
						}
					}
					xs["link"] = lStr
				} else {
					i, err := strconv.Atoi(lStr)
					if err != nil {
						helperError = &HelperError{
							Op:  "ParseTag",
							Tag: sl,
							err: fmt.Errorf("strconv.Atoi failed: %w", err),
						}
						break
					} else {
						xi[sl] = i
					}
				}
			}
		}
	}

	return xb["req"], xi["lenmin"], xi["lenmax"], xs["link"], xb["email"], xi["valmin"], xi["valmax"], xs["regexp"], helperError
}
