package crud

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// Helper reflects the object to generate and cache PostgreSQL queries
// (CREATE TABLE, INSERT, UPDATE etc.), and to setup validation rules for fields
// (min. length, if it is required, regular expression it should match etc.).
// Database table and column names are lowercase with underscore and they are
// generated from field names.
// Field validation is parsed out from the "crud" tag.
// Helper is created within Controller and there is no need to instantiate it
type Helper struct {
	queryDropTable   string
	queryCreateTable string
	queryInsert      string
	queryUpdateById  string
	querySelect      string
	querySelectById  string
	queryDeleteById  string
	dbTbl            string
	dbColPrefix      string
	dbFieldCols      map[string]string
	dbCols           map[string]string
	url              string

	fieldsRequired map[string]bool
	fieldsLength   map[string][2]int
	fieldsEmail    map[string]bool
	fieldsValue    map[string][2]int
	fieldsRegExp   map[string]*regexp.Regexp

	err *HelperError
}

// NewHelper takes object and database table name prefix as arguments and
// returns Helper instance
func NewHelper(obj interface{}, dbTblPrefix string) *Helper {
	h := &Helper{}
	h.reflectStruct(obj, dbTblPrefix)
	return h
}

// Err returns error that occurred when reflecting struct
func (h *Helper) Err() *HelperError {
	return h.err
}

// GetQueryDropTable returns drop table query
func (h Helper) GetQueryDropTable() string {
	return h.queryDropTable
}

// GetQueryCreateTable return create table query
func (h Helper) GetQueryCreateTable() string {
	return h.queryCreateTable
}

// GetQueryInsert returns insert query
func (h *Helper) GetQueryInsert() string {
	return h.queryInsert
}

// GetQueryUpdateById returns update query
func (h *Helper) GetQueryUpdateById() string {
	return h.queryUpdateById
}

// GetQuerySelectById returns select query
func (h *Helper) GetQuerySelectById() string {
	return h.querySelectById
}

// GetQueryDeleteById returns delete query
func (h *Helper) GetQueryDeleteById() string {
	return h.queryDeleteById
}

func (h *Helper) GetQuerySelect(order map[string]string, limit int, offset int, filters map[string]interface{}) string {
	s := h.querySelect

	qOrder := ""
	if order != nil && len(order) > 0 {
		for k, v := range order {
			if h.dbFieldCols[k] == "" && h.dbCols[k] == "" {
				continue
			}

			d := "ASC"
			if v == strings.ToLower("desc") {
				d = "DESC"
			}
			if h.dbFieldCols[k] != "" {
				qOrder = h.addWithComma(qOrder, h.dbFieldCols[k]+" "+d)
			} else {
				qOrder = h.addWithComma(qOrder, k+" "+d)
			}
		}
	}

	qLimitOffset := ""
	if limit > 0 {
		if offset > 0 {
			qLimitOffset = fmt.Sprintf("LIMIT %d OFFSET %d", limit, offset)
		} else {
			qLimitOffset = fmt.Sprintf("LIMIT %d", limit)
		}
	}

	qWhere := ""
	i := 1
	if filters != nil && len(filters) > 0 {
		for k, _ := range filters {
			if h.dbFieldCols[k] == "" {
				continue
			}

			qWhere = h.addWithAnd(qWhere, fmt.Sprintf(h.dbFieldCols[k]+"=$%d", i))
			i++
		}
	}

	if qWhere != "" {
		s += " WHERE " + qWhere
	}
	if qOrder != "" {
		s += " ORDER BY " + qOrder
	}
	if qLimitOffset != "" {
		s += " " + qLimitOffset
	}
	return s
}

func (h *Helper) setFieldFromTag(tag string, fieldIdx int, fieldName string) {
	req, lenmin, lenmax, email, valmin, valmax, re, err := h.parseTag(tag)
	if err != nil {
		h.err = err
		return
	}
	if req {
		h.fieldsRequired[fieldName] = true
	}
	if email {
		h.fieldsEmail[fieldName] = true
	}
	if lenmin > -1 || lenmax > -1 {
		h.fieldsLength[fieldName] = [2]int{lenmin, lenmax}
	}
	if valmin > -1 || valmax > -1 {
		h.fieldsValue[fieldName] = [2]int{valmin, valmax}
	}
	if re != "" {
		h.fieldsRegExp[fieldName] = regexp.MustCompile(re)
	}
}

func (h *Helper) getDBCol(n string) string {
	dbCol := ""
	if n == "ID" {
		dbCol = h.dbColPrefix + "_id"
	} else if n == "Flags" {
		dbCol = h.dbColPrefix + "_flags"
	} else {
		dbCol = h.getUnderscoredName(n)
	}
	return dbCol
}

func (h *Helper) getDBColParams(n string, t string) string {
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

func (h *Helper) addWithComma(s string, v string) string {
	if s != "" {
		s += ","
	}
	s += v
	return s
}

func (h *Helper) addWithAnd(s string, v string) string {
	if s != "" {
		s += " AND "
	}
	s += v
	return s
}

func (h *Helper) reflectStruct(u interface{}, dbTablePrefix string) {
	v := reflect.ValueOf(u)
	i := reflect.Indirect(v)
	s := i.Type()

	usName := h.getUnderscoredName(s.Name())
	usPluName := h.getPluralName(usName)
	h.dbTbl = dbTablePrefix + usPluName
	h.dbColPrefix = usName
	h.url = usPluName

	h.dbFieldCols = make(map[string]string)
	h.dbCols = make(map[string]string)

	h.fieldsRequired = make(map[string]bool)
	h.fieldsLength = make(map[string][2]int)
	h.fieldsValue = make(map[string][2]int)
	h.fieldsEmail = make(map[string]bool)
	h.fieldsRegExp = make(map[string]*regexp.Regexp)

	var queryCreateTableCols, querySelectCols, queryUpdateCols, queryInsertCols, queryInsertVals string
	var updateFieldCnt, insertFieldCnt int

	for j := 0; j < s.NumField(); j++ {
		field := s.Field(j)
		if field.Type.Kind() != reflect.Int64 && field.Type.Kind() != reflect.String && field.Type.Kind() != reflect.Int {
			continue
		}

		h.setFieldFromTag(field.Tag.Get("crud"), j, field.Name)
		if h.err != nil {
			return
		}

		dbCol := h.getDBCol(field.Name)
		h.dbFieldCols[field.Name] = dbCol
		h.dbCols[dbCol] = field.Name
		dbColParams := h.getDBColParams(field.Name, field.Type.String())

		queryCreateTableCols = h.addWithComma(queryCreateTableCols, dbCol+" "+dbColParams)
		querySelectCols = h.addWithComma(querySelectCols, dbCol)
		if field.Name != "ID" {
			updateFieldCnt++
			queryUpdateCols = h.addWithComma(queryUpdateCols, dbCol+"=$"+strconv.Itoa(updateFieldCnt))

			insertFieldCnt++
			queryInsertCols = h.addWithComma(queryInsertCols, dbCol)
			queryInsertVals = h.addWithComma(queryInsertVals, "$"+strconv.Itoa(insertFieldCnt))
		}
	}

	h.queryDropTable = "DROP TABLE IF EXISTS " + h.dbTbl
	h.queryCreateTable = "CREATE TABLE " + h.dbTbl + " (" + queryCreateTableCols + ")"
	h.queryDeleteById = "DELETE FROM " + h.dbTbl + " WHERE " + h.dbColPrefix + "_id = $1"
	h.querySelectById = "SELECT " + querySelectCols + " FROM " + h.dbTbl + " WHERE " + h.dbColPrefix + "_id = $1"
	h.queryInsert = "INSERT INTO " + h.dbTbl + "(" + queryInsertCols + ") VALUES (" + queryInsertVals + ") RETURNING " + h.dbColPrefix + "_id"
	h.queryUpdateById = "UPDATE " + h.dbTbl + " SET " + queryUpdateCols + " WHERE " + h.dbColPrefix + "_id = $" + strconv.Itoa(updateFieldCnt+1)
	h.querySelect = "SELECT " + querySelectCols + " FROM " + h.dbTbl
}

func (h *Helper) getUnderscoredName(s string) string {
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

func (h *Helper) getPluralName(s string) string {
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

func (h *Helper) parseTag(s string) (bool, int, int, bool, int, int, string, *HelperError) {
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
	}
	var helperError *HelperError

	if len(xt) < 1 {
		return xb["req"], xi["lenmin"], xi["lenmax"], xb["email"], xi["valmin"], xi["valmax"], xs["regexp"], helperError
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
				} else {
					i, err := strconv.Atoi(lStr)
					if err != nil {
						helperError = &HelperError{
							Op:  "ParseTag",
							Tag: sl,
							Err: fmt.Errorf("strconv.Atoi failed: %w", err),
						}
						break
					} else {
						xi[sl] = i
					}
				}
			}
		}
	}

	return xb["req"], xi["lenmin"], xi["lenmax"], xb["email"], xi["valmin"], xi["valmax"], xs["regexp"], helperError
}
