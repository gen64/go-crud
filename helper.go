package crud

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
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
	fields      []string

	fieldsRequired     map[string]bool
	fieldsLength       map[string][2]int
	fieldsEmail        map[string]bool
	fieldsValue        map[string][2]int
	fieldsValueNotNil  map[string][2]bool
	fieldsRegExp       map[string]*regexp.Regexp
	fieldsDefaultValue map[string]string
	fieldsUniq         map[string]bool
	fieldsTags         map[string]map[string]string

	fieldsFlags map[string]int

	flags int

	defaultFieldsTags map[string]map[string]string

	err *HelperError
}

const TypeInt64 = 64
const TypeInt = 128
const TypeString = 256

// NewHelper takes object and database table name prefix as arguments and
// returns Helper instance
func NewHelper(obj interface{}, dbTblPrefix string, forceName string, sourceHelper *Helper) *Helper {
	h := &Helper{}
	h.setDefaultTags(sourceHelper)
	h.reflectStruct(obj, dbTblPrefix, forceName)
	return h
}

// Err returns error that occurred when reflecting struct
func (h *Helper) Err() *HelperError {
	return h.err
}

// GetFlags returns flags
func (h *Helper) GetFlags() int {
	return h.flags
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

func (h *Helper) GetQuerySelect(order []string, limit int, offset int, filters map[string]interface{}, orderFieldsToInclude map[string]bool, filterFieldsToInclude map[string]bool) string {
	s := h.querySelectPrefix

	qOrder := ""
	if len(order) > 0 {
		for i := 0; i < len(order); i = i + 2 {
			k := order[i]
			v := order[i+1]

			if len(orderFieldsToInclude) > 0 && !orderFieldsToInclude[k] && !orderFieldsToInclude[h.dbCols[k]] {
				continue
			}

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
	if len(filters) > 0 {
		sorted := []string{}
		for k := range filters {
			if h.dbFieldCols[k] == "" {
				continue
			}
			if len(filterFieldsToInclude) > 0 && !filterFieldsToInclude[k] {
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

func (h *Helper) setDefaultTags(src *Helper) {
	if src != nil {
		h.defaultFieldsTags = make(map[string]map[string]string)
		h.defaultFieldsTags = src.getFieldsTags()
	}
}

// TODO Is it used?
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

func (h *Helper) getFieldsTags() map[string]map[string]string {
	return h.fieldsTags
}

func (h *Helper) reflectStruct(u interface{}, dbTablePrefix string, forceName string) {
	h.reflectStructForValidation(u)
	h.reflectStructForDBQueries(u, dbTablePrefix, forceName)
}

func (h *Helper) reflectStructForDBQueries(u interface{}, dbTablePrefix string, forceName string) {
	v := reflect.ValueOf(u)
	i := reflect.Indirect(v)
	s := i.Type()

	usName := h.getUnderscoredName(s.Name())
	if forceName != "" {
		usName = h.getUnderscoredName(forceName)
	}
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

		if field.Type.Kind() == reflect.Int64 {
			h.fieldsFlags[field.Name] += TypeInt64
		}
		if field.Type.Kind() == reflect.Int {
			h.fieldsFlags[field.Name] += TypeInt
		}
		if field.Type.Kind() == reflect.String {
			h.fieldsFlags[field.Name] += TypeString
		}

		dbCol := h.getDBCol(field.Name)
		h.dbFieldCols[field.Name] = dbCol
		h.dbCols[dbCol] = field.Name
		uniq := false
		if h.fieldsUniq[field.Name] {
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

		h.fields = append(h.fields, field.Name)
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
	h.fieldsFlags = make(map[string]int)
	h.fieldsDefaultValue = make(map[string]string)
	h.fieldsUniq = make(map[string]bool)
	h.fieldsTags = make(map[string]map[string]string)

	for j := 0; j < s.NumField(); j++ {
		field := s.Field(j)
		if field.Type.Kind() != reflect.Int64 && field.Type.Kind() != reflect.String && field.Type.Kind() != reflect.Int {
			continue
		}

		h.setFieldFromName(field.Name)

		h.fieldsLength[field.Name] = [2]int{0, 0}
		h.fieldsValue[field.Name] = [2]int{0, 0}
		h.fieldsValueNotNil[field.Name] = [2]bool{false, false}

		crudTag := field.Tag.Get("crud")
		crudRegexpTag := field.Tag.Get("crud_regexp")
		crudValTag := field.Tag.Get("crud_val")
		if h.defaultFieldsTags != nil {
			if crudTag == "" && h.defaultFieldsTags[field.Name]["crud"] != "" {
				crudTag = h.defaultFieldsTags[field.Name]["crud"]
			}
			if crudRegexpTag == "" && h.defaultFieldsTags[field.Name]["crud_regexp"] != "" {
				crudRegexpTag = h.defaultFieldsTags[field.Name]["crud_regexp"]
			}
			if crudValTag == "" && h.defaultFieldsTags[field.Name]["crud_val"] != "" {
				crudValTag = h.defaultFieldsTags[field.Name]["crud_val"]
			}
		}

		h.setFieldFromTag(crudTag, j, field.Name)
		if h.err != nil {
			return
		}

		if crudRegexpTag != "" {
			h.fieldsRegExp[field.Name] = regexp.MustCompile(crudRegexpTag)
		}
		if crudValTag != "" {
			h.fieldsDefaultValue[field.Name] = crudValTag
		}

		h.fieldsTags[field.Name] = make(map[string]string)
		h.fieldsTags[field.Name]["crud"] = field.Tag.Get("crud")
		h.fieldsTags[field.Name]["crud_regexp"] = field.Tag.Get("crud_regexp")
		h.fieldsTags[field.Name]["crud_val"] = field.Tag.Get("crud_val")
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
					Op:  "ParseTag",
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
		dbColParams = "BIGINT DEFAULT 0"
	} else {
		switch t {
		case "string":
			dbColParams = "VARCHAR(255) DEFAULT ''"
		case "int64":
			dbColParams = "BIGINT DEFAULT 0"
		case "int":
			dbColParams = "BIGINT DEFAULT 0"
		default:
			dbColParams = "VARCHAR(255) DEFAULT ''"
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
