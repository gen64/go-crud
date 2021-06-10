package crud

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"sort"
)

// Helper reflects the object to generate and cache PostgreSQL queries
// (CREATE TABLE, INSERT, UPDATE etc.), and to setup validation rules for fields
// (min. length, if it is required, regular expression it should match etc.).
// Database table and column names are lowercase with underscore and they are
// generated from field names.
// Field validation is parsed out from the "crud" tag.
// Helper is created within Controller and there is no need to instantiate it
type Helper struct {
	queryDropTable    string
	queryCreateTable  string
	queryInsert       string
	queryUpdateById   string
	querySelectById   string
	queryDeleteById   string
	querySelectPrefix string

	dbTbl       string
	dbColPrefix string
	dbFieldCols map[string]string
	dbCols      map[string]string
	url         string

	fieldsRequired map[string]bool
	fieldsLength   map[string][2]int
	fieldsEmail    map[string]bool
	fieldsValue    map[string][2]int
	fieldsValueNotNil map[string][2]bool
	fieldsRegExp   map[string]*regexp.Regexp
	fieldsDefaultValue map[string]string
	fieldsUniq     map[string]bool

	fieldsFlags map[string]int32
	httpFlags   int32

	err *HelperError
}

// Values for fieldsFlags and httpFlags
const HTTPNoRead = 2
const HTTPNoUpdate = 4
const HTTPNoCreate = 8
const HTTPNoDelete = 16
const HTTPNoList = 32

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
func (h *Helper) GetQueryInsert(fields []string) string {
	if len(fields) == 0 {
		return h.queryInsert
	}

	s := "INSERT INTO " + h.dbTbl + "("
	qCols, _, qVals, _ := h.getColsCommaSeparated(fields)
	if qCols == "" {
		return h.queryInsert
	}
	s += qCols
	s += ") VALUES (" + qVals + ") RETURNING " + h.dbColPrefix + "_id"
	return s
}

// GetQueryUpdateById returns update query
func (h *Helper) GetQueryUpdateById(fields []string) string {
	if len(fields) == 0 {
		return h.queryUpdateById
	}

	s := "UPDATE " + h.dbTbl + " SET "
	qCols, cnt, _, qColVals := h.getColsCommaSeparated(fields)
	if qCols == "" {
		return h.queryUpdateById
	}
	s += qColVals + " WHERE " + h.dbColPrefix + "_id = $" + strconv.Itoa(cnt+1)
	return s
}

// GetQuerySelectById returns select query
func (h *Helper) GetQuerySelectById(fields []string) string {
	if len(fields) == 0 {
		return h.querySelectById
	}
	s := "SELECT "
	qCols, _, _, _ := h.getColsCommaSeparated(fields)
	if qCols == "" {
		return h.querySelectById
	}
	s += qCols + " FROM " + h.dbTbl + " WHERE " + h.dbColPrefix + "_id = $1"
	return s
}

// GetQueryDeleteById returns delete query
func (h *Helper) GetQueryDeleteById() string {
	return h.queryDeleteById
}

func (h *Helper) GetQuerySelect(fields []string, order []string, limit int, offset int, filters map[string]interface{}) string {
	s := h.querySelectPrefix
	if len(fields) > 0 {
		s = "SELECT "
		qCols, _, _, _ := h.getColsCommaSeparated(fields)
		if qCols == "" {
			s = h.querySelectPrefix
		} else {
			s += qCols + " FROM " + h.dbTbl
		}
	}

	qOrder := ""
	if order != nil && len(order) > 0 {
		for i:=0; i<len(order); i=i+2 {
			k := order[i]
			v := order[i+1]

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
		sorted := []string{}
		for k, _ := range filters {
			if h.dbFieldCols[k] == "" {
				continue
			}
			sorted = append(sorted, h.dbFieldCols[k])
		}
		sort.Strings(sorted)
		for _, col := range sorted {
			qWhere = h.addWithAnd(qWhere, fmt.Sprintf(col+"=$%d", i))
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

func (h *Helper) getColsCommaSeparated(fields []string) (string, int, string, string) {
	cols := ""
	colCnt := 0
	vals := ""
	colVals := ""
	for _, k := range fields {
		if h.dbFieldCols[k] == "" {
			continue
		}
		cols = h.addWithComma(cols, h.dbFieldCols[k])
		colCnt++
		vals = h.addWithComma(vals, "$"+strconv.Itoa(colCnt))
		colVals = h.addWithComma(colVals, h.dbFieldCols[k]+"=$"+strconv.Itoa(colCnt))
	}
	return cols, colCnt, vals, colVals
}

func (h *Helper) reflectStruct(u interface{}, dbTablePrefix string) {
	h.reflectStructForValidation(u)
	h.reflectStructForDBQueries(u, dbTablePrefix)
	h.reflectStructForHTTP(u)
}

func (h *Helper) reflectStructForDBQueries(u interface{}, dbTablePrefix string) {
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

	colsWithTypes := ""
	cols := ""
	valsWithoutID := ""
	colsWithoutID := ""
	colVals := ""
	idCol := h.dbColPrefix + "_id"

	valCnt := 1
	for j := 0; j < s.NumField(); j++ {
		field := s.Field(j)
		if field.Type.Kind() != reflect.Int64 && field.Type.Kind() != reflect.String && field.Type.Kind() != reflect.Int {
			continue
		}

		dbCol := h.getDBCol(field.Name)
		h.dbFieldCols[field.Name] = dbCol
		h.dbCols[dbCol] = field.Name
		uniq := false
		if h.fieldsUniq[field.Name] == true {
			uniq = true
		}
		dbColParams := h.getDBColParams(field.Name, field.Type.String(), uniq)

		colsWithTypes = h.addWithComma(colsWithTypes, dbCol+" "+dbColParams)
		cols = h.addWithComma(cols, dbCol)

		if field.Name != "ID" {
			colsWithoutID = h.addWithComma(colsWithoutID, dbCol)
			valsWithoutID = h.addWithComma(valsWithoutID, "$"+strconv.Itoa(valCnt))
			colVals = h.addWithComma(colVals, dbCol+"=$"+strconv.Itoa(valCnt))
			valCnt++
		}
	}

	h.queryDropTable = fmt.Sprintf("DROP TABLE IF EXISTS %s", h.dbTbl)
	h.queryCreateTable = fmt.Sprintf("CREATE TABLE %s (%s)", h.dbTbl, colsWithTypes)
	h.queryDeleteById = fmt.Sprintf("DELETE FROM %s WHERE %s = $1", h.dbTbl, idCol)
	h.querySelectById = fmt.Sprintf("SELECT %s FROM %s WHERE %s = $1", cols, h.dbTbl, idCol)
	h.queryInsert = fmt.Sprintf("INSERT INTO %s(%s) VALUES (%s) RETURNING %s", h.dbTbl, colsWithoutID, valsWithoutID, idCol)
	h.queryUpdateById = fmt.Sprintf("UPDATE %s SET %s WHERE %s = $%d", h.dbTbl, colVals, idCol, valCnt)
	h.querySelectPrefix = fmt.Sprintf("SELECT %s FROM %s", cols, h.dbTbl)
}

func (h *Helper) reflectStructForValidation(u interface{}) {
	v := reflect.ValueOf(u)
	i := reflect.Indirect(v)
	s := i.Type()

	h.fieldsRequired = make(map[string]bool)
	h.fieldsLength = make(map[string][2]int)
	h.fieldsValue = make(map[string][2]int)
	h.fieldsValueNotNil = make(map[string][2]bool)
	h.fieldsEmail = make(map[string]bool)
	h.fieldsRegExp = make(map[string]*regexp.Regexp)
	h.fieldsFlags = make(map[string]int32)
	h.fieldsDefaultValue = make(map[string]string)
	h.fieldsUniq = make(map[string]bool)

	for j := 0; j < s.NumField(); j++ {
		field := s.Field(j)
		if field.Type.Kind() != reflect.Int64 && field.Type.Kind() != reflect.String && field.Type.Kind() != reflect.Int {
			continue
		}

		h.setFieldFromName(field.Name)

		h.fieldsLength[field.Name] = [2]int{0, 0}
		h.fieldsValue[field.Name] = [2]int{0, 0}
		h.fieldsValueNotNil[field.Name] = [2]bool{false, false}

		h.setFieldFromTag(field.Tag.Get("crud"), j, field.Name)
		if h.err != nil {
			return
		}

		if field.Tag.Get("crud_regexp") != "" {
			h.fieldsRegExp[field.Name] = regexp.MustCompile(field.Tag.Get("crud_regexp"))
		}
		if field.Tag.Get("crud_val") != "" {
			h.fieldsDefaultValue[field.Name] = field.Tag.Get("crud_val")
		}
	}
}

func (h *Helper) reflectStructForHTTP(u interface{}) {
	v := reflect.ValueOf(u)
	i := reflect.Indirect(v)
	s := i.Type()

	for j := 0; j < s.NumField(); j++ {
		field := s.Field(j)
		if field.Type.Kind() != reflect.Int64 && field.Type.Kind() != reflect.String && field.Type.Kind() != reflect.Int {
			continue
		}
		h.setFieldHTTPFromTag(field.Tag.Get("http"), j, field.Name)
		if h.err != nil {

		}
		if field.Name == "ID" {
			h.setHTTPFromTag(field.Tag.Get("http_endpoint"))
		}
	}
}

func (h *Helper) setFieldFromName(fieldName string) {
	if strings.HasSuffix(fieldName, "Email") {
		h.fieldsEmail[fieldName] = true
	}
}

func (h *Helper) setFieldFromTag(tag string, fieldIdx int, fieldName string) {
	var helperError *HelperError
	opts := strings.SplitN(tag, " ", -1)
	for _, opt := range opts {
		if helperError != nil {
			break
		}
		h.setFieldFromTagOptWithoutVal(opt, fieldIdx, fieldName)
		helperError = h.setFieldFromTagOptWithVal(opt, fieldIdx, fieldName)
		if helperError != nil {
			return
		}
	}
}

func (h *Helper) setFieldFromTagOptWithoutVal(opt string, fieldIdx int, fieldName string) {
	if opt == "req" {
		h.fieldsRequired[fieldName] = true
	}
	if opt == "email" {
		h.fieldsEmail[fieldName] = true
	}
	if opt == "uniq" {
		h.fieldsUniq[fieldName] = true
	}
}

func (h *Helper) setFieldFromTagOptWithVal(opt string, fieldIdx int, fieldName string) *HelperError {
	for _, valOpt := range []string{"lenmin", "lenmax", "valmin", "valmax", "regexp"} {
		if strings.HasPrefix(opt, valOpt+":") {
			val := strings.Replace(opt, valOpt+":", "", 1)
			if valOpt == "regexp" {
				h.fieldsRegExp[fieldName] = regexp.MustCompile(val)
				continue
			}
			i, err := strconv.Atoi(val)
			if err != nil {
				return &HelperError{
					Op: "ParseTag",
					Tag: valOpt,
					Err: fmt.Errorf("strconv.Atoi failed: %w", err),
				}
			}
			switch valOpt {
			case "lenmin":
				h.fieldsLength[fieldName] = [2]int{i, h.fieldsLength[fieldName][1]}
			case "lenmax":
				h.fieldsLength[fieldName] = [2]int{h.fieldsLength[fieldName][0], i}
			case "valmin":
				h.fieldsValue[fieldName] = [2]int{i, h.fieldsValue[fieldName][1]}
				if i == 0 {
					h.fieldsValueNotNil[fieldName] = [2]bool{true, h.fieldsValueNotNil[fieldName][1]}
				}
			case "valmax":
				h.fieldsValue[fieldName] = [2]int{h.fieldsValue[fieldName][0], i}
				if i == 0 {
					h.fieldsValueNotNil[fieldName] = [2]bool{h.fieldsValueNotNil[fieldName][0], true}
				}
			}
		}
	}
	return nil
}

func (h *Helper) setFieldHTTPFromTag(tag string, fieldIdx int, fieldName string) {
	opts := strings.SplitN(tag, " ", -1)
	if len(opts) > 0 {
		for _, v := range opts {
			if v == "nocreate" {
				if h.fieldsFlags[fieldName] & HTTPNoCreate == 0 {
					h.fieldsFlags[fieldName] += HTTPNoCreate
				}
			}
			if v == "noread" {
				if h.fieldsFlags[fieldName] & HTTPNoRead == 0 {
					h.fieldsFlags[fieldName] += HTTPNoRead
				}
			}
			if v == "noupdate" {
				if h.fieldsFlags[fieldName] & HTTPNoUpdate == 0 {
					h.fieldsFlags[fieldName] += HTTPNoUpdate
				}
			}
			if v == "nolist" {
				if h.fieldsFlags[fieldName] & HTTPNoList == 0 {
					h.fieldsFlags[fieldName] += HTTPNoList
				}
			}
		}
	}
}

func (h *Helper) setHTTPFromTag(tag string) {
	opts := strings.SplitN(tag, " ", -1)
	if len(opts) > 0 {
		for _, v := range opts {
			if v == "nocreate" {
				if h.httpFlags & HTTPNoCreate == 0 {
					h.httpFlags += HTTPNoCreate
				}
			}
			if v == "noread" {
				if h.httpFlags & HTTPNoRead == 0 {
					h.httpFlags += HTTPNoRead
				}
			}
			if v == "noupdate" {
				if h.httpFlags & HTTPNoUpdate == 0 {
					h.httpFlags += HTTPNoUpdate
				}
			}
			if v == "nodelete" {
				if h.httpFlags & HTTPNoDelete == 0 {
					h.httpFlags += HTTPNoDelete
				}
			}
			if v == "nolist" {
				if h.httpFlags & HTTPNoList == 0 {
					h.httpFlags += HTTPNoList
				}
			}
		}
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

func (h *Helper) getDBColParams(n string, t string, uniq bool) string {
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
	if uniq {
		dbColParams += " UNIQUE"
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
