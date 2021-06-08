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
	s := h.querySelect
	if len(fields) > 0 {
		s = "SELECT "
		qCols, _, _, _ := h.getColsCommaSeparated(fields)
		if qCols == "" {
			s = h.querySelect
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

func (h *Helper) getColsCommaSeparated(fields []string) (string, int, string, string) {
	c := ""
	i := 0
	v := ""
	cv := ""
	for _, k := range fields {
		if h.dbFieldCols[k] == "" {
			continue
		}
		c = h.addWithComma(c, h.dbFieldCols[k])
		i++
		v = h.addWithComma(v, "$"+strconv.Itoa(i))
		cv = h.addWithComma(cv, h.dbFieldCols[k]+"=$"+strconv.Itoa(i))
	}
	return c, i, v, cv
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

	colsWithTypes := []string{}
	cols := []string{}
	colsWithoutID := []string{}
	idCol := h.dbColPrefix + "_id"

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

		colsWithTypes = append(colsWithTypes, dbCol)
		colsWithTypes = append(colsWithTypes, dbColParams)
		cols = append(cols, dbCol)

		if field.Name != "ID" {
			colsWithoutID = append(colsWithoutID, dbCol)
		}
	}

	h.queryDropTable = h.buildQueryDropTable(h.dbTbl)
	h.queryCreateTable = h.buildQueryCreateTable(h.dbTbl, colsWithTypes)
	h.queryDeleteById = h.buildQueryDeleteById(h.dbTbl, idCol)
	h.querySelectById = h.buildQuerySelectById(h.dbTbl, cols, idCol)
	h.queryInsert = h.buildQueryInsert(h.dbTbl, colsWithoutID, idCol)
	h.queryUpdateById = h.buildQueryUpdateById(h.dbTbl, colsWithoutID, idCol)
	h.querySelect = h.buildQuerySelect(h.dbTbl, cols, []string{}, 0, 0, nil)
}

func (h *Helper) buildQueryDropTable(tbl string) string {
	return fmt.Sprintf("DROP TABLE IF EXISTS %s", tbl)
}

func (h *Helper) buildQueryCreateTable(tbl string, cols []string) string {
	c := ""
	for i:=0; i<len(cols); i=i+2 {
		c = h.addWithComma(c, cols[i] + " " + cols[i+1])
	}
	return fmt.Sprintf("CREATE TABLE %s (%s)", tbl, c)
}

func (h *Helper) buildQueryDeleteById(tbl string, idCol string) string {
	return fmt.Sprintf("DELETE FROM %s WHERE %s = $1", tbl, idCol)
}

func (h *Helper) buildQuerySelectById(tbl string, cols []string, idCol string) string {
	if len(cols) == 0 {
		cols = append(cols, "*")
	}
	c := ""
	for i:=0; i<len(cols); i++ {
		c = h.addWithComma(c, cols[i])
	}
	return fmt.Sprintf("SELECT %s FROM %s WHERE %s = $1", c, tbl, idCol)
}

func (h *Helper) buildQueryInsert(tbl string, cols []string, idCol string) string {
	if len(cols) == 0 {
		return ""
	}

	c := ""
	v := ""
	for i:=0; i<len(cols); i++ {
		c = h.addWithComma(c, cols[i])
		v = h.addWithComma(v, "$"+strconv.Itoa(i+1))
	}
	return fmt.Sprintf("INSERT INTO %s(%s) VALUES (%s) RETURNING %s", tbl, c, v, idCol)
}

func (h *Helper) buildQueryUpdateById(tbl string, cols []string, idCol string) string {
	if len(cols) == 0 {
		return ""
	}

	c := ""
	for i:=0; i<len(cols); i++ {
		c = h.addWithComma(c, cols[i]+"=$"+strconv.Itoa(i+1))
	}
	return fmt.Sprintf("UPDATE %s SET %s WHERE %s = $%d", tbl, c, idCol, len(cols)+1)
}

func (h *Helper) buildQuerySelect(tbl string, cols []string, order []string, limit int, offset int, filters map[string]interface{}) string {
	if len(cols) == 0 {
		cols = append(cols, "*")
	}
	c := ""
	for i:=0; i<len(cols); i++ {
		c = h.addWithComma(c, cols[i])
	}
	q := fmt.Sprintf("SELECT %s FROM %s", c, tbl)

	qOrder := ""
	if len(order) > 0 {
		for i:=0; i<len(order); i=i+2 {
			k := order[i]
			v := order[i+1]
			d := "ASC"
			if v == strings.ToLower("desc") {
				d = "DESC"
			}
			qOrder = h.addWithComma(qOrder, k+" "+d)
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
			qWhere = h.addWithAnd(qWhere, fmt.Sprintf(k+"=$%d", i))
			i++
		}
	}

	if qWhere != "" {
		q += " WHERE " + qWhere
	}
	if qOrder != "" {
		q += " ORDER BY " + qOrder
	}
	if qLimitOffset != "" {
		q += " " + qLimitOffset
	}
	return q
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
