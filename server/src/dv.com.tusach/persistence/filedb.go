package persistence

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"dv.com.tusach/logger"
	"dv.com.tusach/model"
	"dv.com.tusach/util"
)

type FileDB struct {
	info         model.SystemInfo
	users        []model.User
	books        []model.Book
	deletedBooks []model.Book
	drive        *GoogleDriveStore
}

var chapterNoPattern = regexp.MustCompile(`(?P<first>playOrder=")\d*(?P<second>")`)
var chapterTitlePattern = regexp.MustCompile(`(?P<a>\Q<navLabel><text>\E)[^<]+(?P<b>\Q<\E)`)

func (f *FileDB) InitDB() {
	f.info.SystemUpTime = util.TimestampNow()
	f.initUsers()
	f.books = []model.Book{}
	f.drive = &GoogleDriveStore{}
	f.loadLocalBooks()
	f.drive.Init()
	if f.drive.tusachFolderId != "" {
		f.loadDriveBooks()
	}
}

func (f *FileDB) CloseDB() {
}

func (f *FileDB) loadDriveBooks() {
	// load books from google drive
	driveBooks, err := f.drive.LoadBooks()
	if err != nil {
		return
	}

	for _, driveBook := range driveBooks {
		// check timestamp
		index := -1
		for i, localBook := range f.books {
			if localBook.Id == driveBook.Book.Id {
				index = i
				break
			}
		}
		if index == -1 || util.Timestamp2UnixTime(driveBook.Book.LastUpdatedTime) > util.Timestamp2UnixTime(f.books[index].LastUpdatedTime) {
			// download epub file
			logger.Infof("downloading google drive book: %v\n", *driveBook.EpubFile)
			epubData, err := f.drive.DownloadFile(driveBook.EpubFile)
			if err == nil {
				// save the json/epub file
				filename := GetBookEpubFilename(*driveBook.Book)
				logger.Infof("writing file %s\n", filename)
				if err := ioutil.WriteFile(filename, epubData, 0644); err != nil {
					errMsg := fmt.Sprintf("failed to write book %s: %s", filename, err.Error())
					logger.Error(errMsg)
					panic(errMsg)
				}

				filename = GetBookMetaFilename(*driveBook.Book)
				logger.Infof("writing file %s\n", filename)
				if err := util.WriteJsonFile(filename, driveBook.Book); err != nil {
					errMsg := fmt.Sprintf("failed to write book %s: %s", filename, err.Error())
					logger.Error(errMsg)
					panic(errMsg)
				}
				if index != -1 {
					logger.Infof("replace with google drive book: %v\n", *driveBook.Book)
					f.books[index] = *driveBook.Book
				} else {
					logger.Infof("added google drive book: %v\n", *driveBook.Book)
					f.books = append(f.books, *driveBook.Book)
				}
			} else {
				logger.Errorf("failed to download epub media: %s(%s). %v\n", driveBook.EpubFile.FileId, driveBook.EpubFile.FileName, err)
			}
		}
	}
}

func (f *FileDB) loadLocalBooks() {
	logger.Info("reading books from local file system...")
	// load books
	booksPath := filepath.Join(util.GetConfiguration().LibraryPath, "books")
	list, err := util.ReadDir(booksPath, true)
	if err != nil {
		panic(fmt.Sprintf("failed to read book directory %s: %s", booksPath, err.Error()))
	}
	for _, name := range list {
		if strings.HasSuffix(name, ".epub") {
			// find the corresponding .json file
			jsonfile := strings.Replace(name, ".epub", ".json", 1)
			if util.ContainsString(list, jsonfile) {
				fullname := filepath.Join(booksPath, jsonfile)
				logger.Infof("Loading book meta: %s", fullname)
				book := model.Book{}
				err = util.ReadJsonFile(fullname, &book)
				if err != nil {
					logger.Errorf("Failed to load book meta %s", err)
				} else {
					logger.Infof("Loaded book: +v", book)
					f.books = append(f.books, book)
				}
			}
		}
	}

	// reset all books in progress to aborted
	for _, book := range f.books {
		updated := false
		if book.LastUpdatedTime == nil {
			book.LastUpdatedTime = util.TimestampNow()
			updated = true
		}
		if book.Status == model.BookStatusType_IN_PROGRESS {
			book.Status = model.BookStatusType_ABORTED
			updated = true
		}
		if updated {
			f.SaveBook(&book)
		}
	}
	logger.Infof("Loaded books: +v", f.books)
}

func (f *FileDB) initUsers() {
	logger.Info("initialise users...")

	filename := filepath.Join(util.GetConfiguration().LibraryPath, "users.json")
	f.users = []model.User{}
	err := util.ReadJsonFile(filename, f.users)
	if err != nil || len(f.users) == 0 {
		logger.Info("creating default users...")
		err = f.SaveUser(model.User{Name: "admin", Roles: "administrator"})
		if err != nil {
			panic("Error saving user! " + err.Error())
		}
		err = f.SaveUser(model.User{Name: "vinhvan", Roles: "user"})
		if err != nil {
			panic("Error saving user! " + err.Error())
		}
		err = f.SaveUser(model.User{Name: "guest", Roles: "user"})
		if err != nil {
			panic("Error saving user! " + err.Error())
		}
	}
	logger.Infof("loaded users: %v", f.users)
}

func (f *FileDB) GetSystemInfo() model.SystemInfo {
	return f.info
}

func (f *FileDB) SaveSystemInfo(info model.SystemInfo) {
	logger.Debugf("save systemInfo=%v", info)
	f.info.SystemUpTime = info.SystemUpTime
	f.info.BookLastUpdatedTime = info.BookLastUpdatedTime
}

func (f *FileDB) GetUsers() []model.User {
	return f.users
}

func (f *FileDB) SaveUser(user model.User) error {
	filename := filepath.Join(util.GetConfiguration().LibraryPath, "users.json")
	index := -1
	for i := 0; i < len(f.users); i++ {
		if f.users[i].Name == user.Name {
			index = i
			break
		}
	}
	if index != -1 {
		f.users[index] = user
	} else {
		f.users = append(f.users, user)
	}
	return util.WriteJsonFile(filename, f.users)
}

func (f *FileDB) DeleteUser(userName string) error {
	filename := filepath.Join(util.GetConfiguration().LibraryPath, "users.json")
	for i := 0; i < len(f.users); i++ {
		if f.users[i].Name == userName {
			logger.Infof("deleting user: %s", userName)
			f.users = append(f.users[:i], f.users[i+1:]...)
			break
		}
	}
	return util.WriteJsonFile(filename, f.users)
}

func (f *FileDB) GetBook(id int32) model.Book {
	for i := 0; i < len(f.books); i++ {
		if f.books[i].Id == id {
			return f.books[i]
		}
	}
	return model.Book{}
}

func (f *FileDB) GetBooks(includeDeleted bool) []model.Book {
	books := f.books
	if includeDeleted && len(f.deletedBooks) > 0 {
		books = append(books, f.deletedBooks...)
	}
	return books
}

func (f *FileDB) GetMaxBookId() (int32, error) {
	var maxBookId int32 = 0
	for i := 0; i < len(f.books); i++ {
		if f.books[i].Id > maxBookId {
			maxBookId = f.books[i].Id
		}
	}
	// check the book id from the epub files
	booksPath := filepath.Join(util.GetConfiguration().LibraryPath, "books")
	list, err := util.ReadDir(booksPath, true)
	if err != nil {
		logger.Errorf("failed to list directory %s: %s", booksPath, err.Error())
		return 0, err
	}
	logger.Infof("found %d books in %s", len(list), booksPath)
	for _, name := range list {
		if strings.HasSuffix(name, ".epub") && len(name) > 8 {
			n, err := strconv.Atoi(name[0:8])
			logger.Infof("parsing book: %s - id:%d", name, n)
			if err == nil && int32(n) > maxBookId {
				maxBookId = int32(n)
			}
		}
	}
	return maxBookId, nil
}

func (f *FileDB) ReplaceBook(book *model.Book) error {
	logger.Infof("Replacing book: %v", book)
	foundIndex := -1
	for i := 0; book.Id > 0 && i < len(f.books); i++ {
		if f.books[i].Id == book.Id {
			foundIndex = i
			break
		}
	}
	if foundIndex != -1 {
		f.books[foundIndex] = *book
	}
	// write to the file
	filename := GetBookMetaFilename(*book)
	err := util.WriteJsonFile(filename, book)
	return err
}

func (f *FileDB) SaveBook(book *model.Book) (retId int, retErr error) {
	logger.Infof("Saving book: %v", book)
	foundIndex := -1
	for i := 0; book.Id > 0 && i < len(f.books); i++ {
		if f.books[i].Id == book.Id {
			foundIndex = i
			break
		}
	}
	if foundIndex != -1 {
		f.books[foundIndex] = *book
	} else {
		maxBookId, err := f.GetMaxBookId()
		if err != nil {
			return 0, err
		}
		book.Id = maxBookId + 1
		f.books = append(f.books, *book)
	}
	book.LastUpdatedTime = util.TimestampNow()

	// write to the file
	filename := GetBookMetaFilename(*book)
	err := util.WriteJsonFile(filename, book)
	f.SaveSystemInfo(model.SystemInfo{BookLastUpdatedTime: book.LastUpdatedTime, SystemUpTime: f.info.SystemUpTime})

	// save to google drive
	if (f.drive != nil && f.drive.tusachFolderId != "") && (book.Status == model.BookStatusType_ABORTED ||
		book.Status == model.BookStatusType_ERROR || book.Status == model.BookStatusType_COMPLETED) {
		f.drive.StoreBook(*book)
	}

	return int(book.Id), err
}

func (f *FileDB) DeleteBook(bookId int32) error {
	logger.Infof("Deleting book ID=" + strconv.Itoa(int(bookId)))
	foundIndex := -1
	for i := 0; i < len(f.books); i++ {
		if f.books[i].Id == bookId {
			foundIndex = i
			break
		}
	}
	if foundIndex != -1 {
		book := f.books[foundIndex]
		// add to deleted books
		book.Deleted = true
		book.LastUpdatedTime = util.TimestampNow()
		f.deletedBooks = append(f.deletedBooks, book)

		RemoveBookDir(book)
		f.books = append(f.books[:foundIndex], f.books[foundIndex+1:]...)
		err := os.Remove(GetBookMetaFilename(book))
		if err != nil {
			return err
		}
		err = os.Remove(GetBookEpubFilename(book))
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("Book " + strconv.Itoa(int(bookId)) + " not found")
}

func (f *FileDB) GetChapters(bookId int32) ([]model.Chapter, error) {
	// read chapters from toc file
	tocFile := filepath.Join(GetBookPath(int(bookId)), "/OEBPS/toc.ncx")
	data, err := ioutil.ReadFile(tocFile)
	if err != nil {
		return nil, errors.New("Failed to open file: " + tocFile + ". " + err.Error())
	}
	chapters := []model.Chapter{}
	lines := strings.Split(string(data), "\n")
	for i := 0; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "<navPoint") && strings.Contains(lines[i], "playOrder=\"") &&
			(i < len(lines)-1) && strings.HasPrefix(lines[i+1], "<navLabel><text>") {
			chapter := model.Chapter{BookId: bookId}
			loc := chapterNoPattern.FindIndex([]byte(lines[i]))
			if len(loc) == 2 && loc[1] > len("playOrder=\"") {
				chapterNo, _ := strconv.Atoi(lines[i][loc[len("playOrder=\"")]:loc[1]])
				chapter.ChapterNo = int32(chapterNo)
			}
			loc = chapterTitlePattern.FindIndex([]byte(lines[i+1]))
			if len(loc) == 2 && loc[1] > len("<navLabel><text>") {
				chapter.Title = lines[i][loc[len("<navLabel><text>")]:loc[1]]
			}
			logger.Infof("found chapter +v", chapter)
			chapters = append(chapters, chapter)
		}
	}

	if len(chapters) == 0 {
		logger.Warnf("No chapter found!\n")
	} else {
		// sort chapters by ChapterNo
		sort.Sort(model.ByChapterNo(chapters))
	}

	// TODO verify chapter html/images from file system
	return chapters, nil
}

func (f *FileDB) SaveChapter(chapter model.Chapter) error {
	book := f.GetBook(chapter.BookId)
	if book.Id == 0 {
		return errors.New("Book " + strconv.Itoa(int(chapter.BookId)) + " not found!")
	}
	return WriteChapter(book, chapter)
}
