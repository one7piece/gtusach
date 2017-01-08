package persistence

import (
	"database/sql"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"
	"sort"
	"strings"
	"time"

	"dv.com.tusach/logger"
	"dv.com.tusach/maker"
	"dv.com.tusach/util"
	_ "github.com/mattn/go-sqlite3"
)

var systemInfo maker.SystemInfo

type Sqlite3 struct {
	db   *sql.DB
	info maker.SystemInfo
}

func (sqlite *Sqlite3) InitDB() {
	sqlite.db = nil
	logger.Infof("opening database %s\n", util.GetConfiguration().DBFilename)
	_db, err := sql.Open("sqlite3", util.GetConfiguration().DBFilename)
	if err != nil {
		logger.Error("failed to open databse: " + util.GetConfiguration().DBFilename + ": " + err.Error())
		panic(err)
	}
	sqlite.db = _db

	// create table systeminfo, datetime store as TEXT (ISO8601 string)
	sqlite.createTable("systeminfo", reflect.TypeOf(maker.SystemInfo{}))
	sqlite.createTable("user", reflect.TypeOf(maker.User{}))
	sqlite.createTable("book", reflect.TypeOf(maker.Book{}))
	sqlite.createTable("chapter", reflect.TypeOf(maker.Chapter{}))

	// init system info
	sqlite.info = maker.SystemInfo{SystemUpTime: time.Now(), BookLastUpdateTime: time.Now(), ParserEditing: false}
	err = sqlite.SaveSystemInfo(sqlite.info)
	if err != nil {
		panic("Error saving system info! " + err.Error())
	}
}

func (sqlite *Sqlite3) CloseDB() {
	if sqlite.db != nil {
		sqlite.db.Close()
	}
}

func (sqlite *Sqlite3) createTable(tableName string, tableType reflect.Type) {
	stmt := "create table if not exists " + tableName + " ("
	for i := 0; i < tableType.NumField(); i++ {
		field := tableType.Field(i)
		persist := isPersistentField(tableType, field.Name)
		if persist {
			colName := field.Name
			//logger.Infof("parsing field: %s:%s\n", field.Name, field.Type.Name())
			var colType string
			switch field.Type.Kind() {
			case reflect.Int:
				colType = "int"
			case reflect.Bool:
				colType = "int"
			default:
				colType = "text"
			}
			if i > 0 {
				stmt += ", "
			}
			stmt += colName + " " + colType
		}
	}
	stmt += ")"
	logger.Infof("executing creating table query: %s\n", stmt)
	_, err := sqlite.db.Exec(stmt)
	if err != nil {
		logger.Infof("failed to create table: %s\n", err)
	}
}

func (sqlite *Sqlite3) GetSystemInfo(forceReload bool) (maker.SystemInfo, error) {
	if forceReload {
		records, err := sqlite.loadRecords(reflect.TypeOf(maker.SystemInfo{}), "systeminfo", "", nil)
		if err != nil {
			return maker.SystemInfo{}, err
		}
		if len(records) > 0 {
			sqlite.info = records[0].(maker.SystemInfo)
			logger.Infof("Found systeminfo: %+v\n", sqlite.info)
		} else {
			logger.Infof("No systeminfo found\n")
		}
	}
	return sqlite.info, nil
}

func (sqlite *Sqlite3) SaveSystemInfo(info maker.SystemInfo) error {
	records, err := sqlite.loadRecords(reflect.TypeOf(maker.SystemInfo{}), "systeminfo", "", nil)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		// insert
		sqlite.insertRecord("systeminfo", reflect.ValueOf(info))
	} else {
		// update
		sqlite.updateRecord("systeminfo", reflect.ValueOf(info), "", nil)
	}
	return nil
}

func (sqlite *Sqlite3) LoadUsers() ([]maker.User, error) {
	records, err := sqlite.loadRecords(reflect.TypeOf(maker.User{}), "user", "", nil)
	if err != nil {
		return []maker.User{}, err
	}
	users := []maker.User{}
	for i := 0; i < len(records); i++ {
		user := records[i].(maker.User)
		users = append(users, user)
		logger.Infof("Found user: %+v\n", user)
	}
	if len(users) == 0 {
		logger.Infof("No users found\n")
	}
	return users, nil
}

func (sqlite *Sqlite3) SaveUser(user maker.User) error {
	args := []interface{}{user.Name}
	records, err := sqlite.loadRecords(reflect.TypeOf(maker.User{}), "user", "Name=?", args)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		// insert
		sqlite.insertRecord("user", reflect.ValueOf(user))
	} else {
		// update
		sqlite.updateRecord("user", reflect.ValueOf(user), "Name=?", args)
	}
	return nil
}

func (sqlite *Sqlite3) DeleteUser(userName string) error {
	args := []interface{}{userName}
	err := sqlite.deleteRecords("user", "Name=?", args)
	return err
}

func (sqlite *Sqlite3) LoadBook(id int) (maker.Book, error) {
	args := []interface{}{id}
	records, err := sqlite.loadRecords(reflect.TypeOf(maker.Book{}), "book", "ID=?", args)
	if err != nil {
		return maker.Book{}, err
	}
	if len(records) > 0 {
		return records[0].(maker.Book), nil
	}
	return maker.Book{}, nil
}

func (sqlite *Sqlite3) LoadBooks() ([]maker.Book, error) {
	records, err := sqlite.loadRecords(reflect.TypeOf(maker.Book{}), "book", "", nil)
	if err != nil {
		return []maker.Book{}, err
	}
	books := []maker.Book{}
	for i := 0; i < len(records); i++ {
		book := records[i].(maker.Book)
		books = append(books, book)
		logger.Infof("Found book: %+v\n", book)
	}
	if len(books) == 0 {
		logger.Infof("No books found\n")
	}
	return books, nil
}

func (sqlite *Sqlite3) SaveBook(book maker.Book) (retId int, retErr error) {

	var newBookId = 0
	defer func() {
		if err := recover(); err != nil {
			//logger.Infof("Recover from panic: %s\n", err)
			if newBookId > 0 {
				sqlite.DeleteBook(newBookId)
			}
			retErr = util.ExtractError(err)
			/*
				// find out what exactly is err
				switch x := err.(type) {
				case string:
					retErr = errors.New(x)
				case error:
					retErr = x
				default:
					retErr = errors.New("Unknow panic")
				}
			*/
			retId = 0
		}
	}()

	// TODO need locking here

	book.LastUpdateTime = time.Now()
	if book.ID == 0 {
		rows, retErr := sqlite.db.Query("SELECT max(ID) FROM book")
		if retErr != nil {
			return 0, retErr
		}
		defer rows.Close()

		if rows.Next() {
			var maxId int
			rows.Scan(&maxId)
			newBookId = maxId + 1
		}
		rows.Close()

		if newBookId == 0 {
			newBookId = 1
		}
		// insert
		book.ID = newBookId
		retErr = sqlite.insertRecord("book", reflect.ValueOf(book))
		if retErr == nil {
			// create book dir
			dirPath := util.GetBookPath(book.ID)
			logger.Infof("Creating book dir: ", dirPath)
			os.MkdirAll(dirPath, 0777)
			if _, err := os.Stat(dirPath); os.IsNotExist(err) {
				panic("Error creating directory: " + dirPath)
			}
			files, err := ioutil.ReadDir(util.GetEpubPath())
			if err != nil {
				panic("Error reading directory: " + util.GetEpubPath() + ". " + err.Error())
			}
			for _, file := range files {
				cmd := exec.Command("cp", "-rf", util.GetEpubPath()+"/"+file.Name(), dirPath)
				out, retErr := cmd.CombinedOutput()
				if retErr != nil {
					panic("Error copying epub template file: " + util.GetEpubPath() + "/" + file.Name() + ". " + string(out))
				}
			}
		}
	} else {
		// update
		args := []interface{}{book.ID}
		retErr = sqlite.updateRecord("book", reflect.ValueOf(book), "ID=?", args)
	}

	systemInfo.BookLastUpdateTime = book.LastUpdateTime
	sqlite.SaveSystemInfo(systemInfo)

	return book.ID, retErr
}

func (sqlite *Sqlite3) DeleteBook(bookId int) error {
	logger.Infof("Deleting book: ", bookId)
	// TODO need locking here

	// delete all chapters of book
	args := []interface{}{bookId}
	err := sqlite.deleteRecords("chapter", "bookId=?", args)
	if err != nil {
		return err
	}

	// remove book
	args = []interface{}{bookId}
	err = sqlite.deleteRecords("book", "ID=?", args)

	// remove files
	err = os.RemoveAll(util.GetBookPath(bookId))

	systemInfo.BookLastUpdateTime = time.Now()
	sqlite.SaveSystemInfo(systemInfo)

	return err
}

func (sqlite *Sqlite3) LoadChapters(bookId int) ([]maker.Chapter, error) {
	var records []interface{}
	var err error
	if bookId > 0 {
		args := []interface{}{bookId}
		records, err = sqlite.loadRecords(reflect.TypeOf(maker.Chapter{}), "chapter", "BookId=?", args)
	} else {
		records, err = sqlite.loadRecords(reflect.TypeOf(maker.Chapter{}), "chapter", "", nil)
	}
	if err != nil {
		return []maker.Chapter{}, err
	}

	chapters := []maker.Chapter{}
	for i := 0; i < len(records); i++ {
		chapter := records[i].(maker.Chapter)
		chapters = append(chapters, chapter)
		//logger.Infof("Found chapter: %+v\n", chapter)
	}
	if len(chapters) == 0 {
		logger.Infof("No chapter found\n")
	} else {
		// sort chapters by ChapterNo
		sort.Sort(maker.ByChapterNo(chapters))
	}

	// TODO verify chapter html/images from file system

	return chapters, nil
}

func (sqlite *Sqlite3) SaveChapter(chapter maker.Chapter) error {
	/*
		filepath := util.GetChapterFilename(chapter.BookId, chapter.ChapterNo)
		err := ioutil.WriteFile(filepath, chapter.Html, 0777)
		if err != nil {
			logger.Infof("error writing chapter file: ", filepath, err)
			return err
		}
	*/
	args := []interface{}{chapter.BookId, chapter.ChapterNo}
	records, err := sqlite.loadRecords(reflect.TypeOf(maker.Chapter{}), "chapter", "BookId=? and ChapterNo=?", args)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		// save record
		err = sqlite.insertRecord("chapter", reflect.ValueOf(chapter))
	} else {
		// save record
		err = sqlite.updateRecord("chapter", reflect.ValueOf(chapter), "BookId=? and ChapterNo=?", args)
	}

	return err
}

func (sqlite *Sqlite3) loadRecords(tableType reflect.Type, tableName string, whereStr string, args []interface{}) ([]interface{}, error) {
	fieldNames := []string{}
	for i := 0; i < tableType.NumField(); i++ {
		field := tableType.Field(i)
		persist := isPersistentField(tableType, field.Name)
		if persist {
			fieldNames = append(fieldNames, field.Name)
		}
	}

	query := "SELECT " + strings.Join(fieldNames, ",") + " FROM " + tableName
	if whereStr != "" {
		query += " WHERE " + whereStr
	}
	logger.Infof("executing query: %s\n", query)
	rows, err := sqlite.db.Query(query, args...)
	if err != nil {
		logger.Errorf("Error executing query. ", err)
		return nil, err
	}
	defer rows.Close()

	records := []interface{}{}
	colValues := make([]interface{}, len(fieldNames))
	colValuePtrs := make([]interface{}, len(fieldNames))

	for rows.Next() {
		// create object of type tableType
		recordValue := reflect.New(tableType).Elem()
		for i := 0; i < len(fieldNames); i++ {
			colValuePtrs[i] = &colValues[i] // store address of value
		}
		rows.Scan(colValuePtrs...)

		for i := 0; i < tableType.NumField(); i++ {
			field := tableType.Field(i)
			if isPersistentField(tableType, field.Name) {
				var colval interface{}
				// colValues hold array of bytes
				byteArr, ok := colValues[i].([]byte)
				if ok {
					colval = string(byteArr)
				} else {
					colval = colValues[i]
				}
				//logger.Infof("scan col field: %s=%v\n", field.Name, colval)
				v := db2field(field.Type, colval)
				recordValue.FieldByName(field.Name).Set(reflect.ValueOf(v))
			}
		}
		records = append(records, recordValue.Interface())
	}
	return records, nil
}

func (sqlite *Sqlite3) insertRecord(tableName string, value reflect.Value) error {
	tx, err := sqlite.db.Begin()
	if err != nil {
		logger.Infof("failed to start transaction.", err)
		return err
	}

	nameStr := ""
	valueStr := ""
	params := []interface{}{}
	for i := 0; i < value.NumField(); i++ {
		persist := isPersistentField(value.Type(), value.Type().Field(i).Name)
		if persist {
			if len(nameStr) > 0 {
				nameStr += ","
			}
			nameStr += value.Type().Field(i).Name
			if len(valueStr) > 0 {
				valueStr += ","
			}
			valueStr += "?"
			v := field2db(value.Type().Field(i).Type, value.Field(i).Interface())
			params = append(params, v)
		}
	}

	insertStr := "INSERT INTO " + tableName + "(" + nameStr + ") values(" + valueStr + ")"

	pstmt, err := tx.Prepare(insertStr)
	if err != nil {
		logger.Infof("failed to prepare insert.", err)
		return err
	}
	defer pstmt.Close()

	logger.Infof("executing insert: ", insertStr)
	_, err = pstmt.Exec(params...)
	if err != nil {
		logger.Infof("failed to execute insert.", err)
		return err
	}

	tx.Commit()
	return nil
}

func (sqlite *Sqlite3) updateRecord(tableName string, value reflect.Value, whereStr string, whereArgs []interface{}) error {
	tx, err := sqlite.db.Begin()
	if err != nil {
		logger.Infof("failed to start transaction.", err)
		return err
	}

	updateStr := "UPDATE " + tableName + " SET "
	params := []interface{}{}
	for i := 0; i < value.NumField(); i++ {
		persist := isPersistentField(value.Type(), value.Type().Field(i).Name)
		if persist {
			if i > 0 {
				updateStr += ","
			}
			updateStr += value.Type().Field(i).Name + "=?"

			v := field2db(value.Type().Field(i).Type, value.Field(i).Interface())
			params = append(params, v)
		}
	}
	if whereStr != "" {
		updateStr += " WHERE " + whereStr
		params = append(params, whereArgs...)
	}

	pstmt, err := tx.Prepare(updateStr)
	if err != nil {
		logger.Infof("failed to prepare update.", err)
		return err
	}
	defer pstmt.Close()

	logger.Infof("executing update: ", updateStr)
	_, err = pstmt.Exec(params...)
	if err != nil {
		logger.Infof("failed to execute update.", err)
		return err
	}

	tx.Commit()
	return nil
}

func (sqlite *Sqlite3) deleteRecords(tableName string, whereStr string, whereArgs []interface{}) error {
	if whereStr == "" {
		return errors.New("Missing where string!")
	}

	tx, err := sqlite.db.Begin()
	if err != nil {
		logger.Infof("failed to start transaction.", err)
		return err
	}

	deleteStr := "DELETE FROM " + tableName
	if whereStr != "all" {
		deleteStr += " WHERE " + whereStr
	}

	pstmt, err := tx.Prepare(deleteStr)
	if err != nil {
		logger.Infof("failed to prepare delete.", err)
		return err
	}
	defer pstmt.Close()

	logger.Infof("executing delete: ", deleteStr)
	_, err = pstmt.Exec(whereArgs...)
	if err != nil {
		logger.Infof("failed to execute delete.", err)
		return err
	}

	tx.Commit()
	return nil
}
