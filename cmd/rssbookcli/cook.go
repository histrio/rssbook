
func main() {
	loggers.InitLoggers(os.Stdout, os.Stdout, os.Stdout, os.Stderr)
	loggers.Info.Printf(
		"Starting ...\ncommit: %s, build time: %s, release: %s",
		version.Commit, version.BuildTime, version.Release,
	)
	var dst string
	var src string
	var bookID string
	var bookTitle string
	var bookAuthor string

	flag.StringVar(&dst, "dst", "", "Generated files destination")
	flag.StringVar(&src, "src", "", "Source of audiofiles")
	flag.StringVar(&bookID, "name", "", "Set a shortname for the podcast. By default it would be a slugifyed source folder name.")
	flag.StringVar(&bookTitle, "title", "", "Set title for the podcast. By default it would take a title from the first file of the book.")
	flag.StringVar(&bookAuthor, "author", "", "Set an author for the podcast. By default it would take an artist from the first file of the book.")
	flag.Parse()

	if src == "" {
		loggers.Warning.Fatalln("No source found.")
	}

	//pwd, err := os.Getwd()
	//utils.Check(err)

	if dst == "" {
		dst = src
		loggers.Warning.Println("No destination specified. '" + src + "' used")
	}

	if bookID == "" {
		bookID = slug.Make(filepath.Base(src))
		loggers.Warning.Println("No book-id specified. '" + bookID + "' used")
	}

	dest := path.Join(dst, bookID)
	err = os.Mkdir(dest, 0777)
	utils.Check(err)

	_title, _author := getTitleAndAuthor(src)
	if bookAuthor == "" {
		bookAuthor = _author
		loggers.Warning.Println("No book author specified. '" + bookAuthor + "' used")
	}
	if bookTitle == "" {
		bookTitle = _title
		loggers.Warning.Println("No book author specified. '" + bookTitle + "' used")
	}

	book := utils.BookMeta{
		ID:     bookID,
		Title:  bookTitle,
		Author: bookAuthor,
	}


	cookRss(book, dest)
	cookM3U(book, dest)
}
