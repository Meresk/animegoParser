package main

import (
	"flag"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/playwright-community/playwright-go"
	"github.com/xuri/excelize/v2"
	"log"
	"os"
	"strings"
	"time"
)

func main() {
	// Допустимые значения для аниме листа
	allowedListTypes := map[string]bool{
		"all":        true,
		"completed":  true,
		"watching":   true,
		"onhold":     true,
		"planned":    true,
		"dropped":    true,
		"rewatching": true,
	}

	user := flag.String("user", "", "animego user name")
	listType := flag.String("type", "all", "animego list type")
	outputFormat := flag.String("output", "excel", "output format (excel/txt)")
	flag.Parse()

	if len(*user) == 0 {
		log.Fatal("user is required")
	}
	if !allowedListTypes[*listType] {
		log.Println("list type is invalid, it will be all")
		*listType = "all"
	}
	if outputFormat == nil || *outputFormat != "excel" && *outputFormat != "txt" {
		log.Println("output format is invalid, it will be excel")
	}

	var url string
	if *listType == "all" {
		url = "https://animego.me/user/" + *user + "/mylist/anime"
	} else {
		url = "https://animego.me/user/" + *user + "/mylist/anime" + fmt.Sprintf("/%s", *listType)
	}

	log.Println(url)
	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("could not launch playwright: %v", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		log.Fatalf("could not launch browser: %v", err)
	}
	defer browser.Close()

	page, err := browser.NewPage()
	if err != nil {
		log.Fatalf("could not create page: %v", err)
	}
	defer page.Close()

	if _, err := page.Goto(url); err != nil {
		log.Fatalf("could not goto: %v", err)
	}

	lastHeight, err := page.Evaluate("document.body.scrollHeight", nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	for {
		// Скроллим страницу вниз
		_, err = page.Evaluate("window.scrollTo(0, document.body.scrollHeight);", nil)
		if err != nil {
			fmt.Println(err)
			return
		}

		time.Sleep(1 * time.Second)

		// Проверяем, достигли ли мы конца страницы
		newHeight, err := page.Evaluate("document.body.scrollHeight", nil)
		if err != nil {
			fmt.Println(err)
			return
		}
		if newHeight.(int) == lastHeight.(int) {
			break
		}
		lastHeight = newHeight
	}

	// Извлечение данных из таблицы
	content, err := page.Content()
	if err != nil {
		log.Fatalf("could not get page content: %v", err)
	}

	// Используем goquery для парсинга контента
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		log.Fatalf("could not parse page content: %v", err)
	}

	switch *outputFormat {
	case "txt":
		file, err := os.Create(*listType + "_anime_list.txt")
		if err != nil {
			log.Fatalf("could not create file: %v", err)
		}
		defer file.Close()

		doc.Find("table tr").Each(func(i int, s *goquery.Selection) {
			s.Find("td[class=\"text-left table-100\"]").Each(func(i int, s *goquery.Selection) {
				s.Find("div[class=\"text-gray-dark-6 small\"]").Each(func(i int, s *goquery.Selection) {
					file.WriteString("Оригинальное название: " + strings.TrimSpace(s.Text()) + "\n")
				})
				s.Find("a").Each(func(i int, s *goquery.Selection) {
					file.WriteString("Русское название: " + strings.TrimSpace(s.Text()) + "\n")
				})
			})

			s.Find("td[data-label=\"Тип\"]").Each(func(i int, s *goquery.Selection) {
				file.WriteString("Тип: " + strings.TrimSpace(s.Text()) + "\n\n")
			})
		})
		log.Println(fmt.Sprintf("Your data in %s_anime_list.txt!", *listType))

	case "excel":
		// Запись в Excel
		file := excelize.NewFile()

		// Добавление заголовков в Excel
		sheet := "Sheet1"
		file.NewSheet(sheet)
		file.SetCellValue(sheet, "A1", "Anime Name")
		file.SetCellValue(sheet, "B1", "Type")

		row := 2
		doc.Find("table tr").Each(func(i int, s *goquery.Selection) {
			// Извлекаем данные и записываем в Excel
			s.Find("td[class=\"text-left table-100\"]").Each(func(i int, s *goquery.Selection) {
				// Извлекаем текст из <div>
				s.Find("div").Each(func(i int, s *goquery.Selection) {
					file.SetCellValue(sheet, fmt.Sprintf("A%d", row), strings.TrimSpace(s.Text()))
				})
				// Извлекаем текст из <a>
				s.Find("a").Each(func(i int, s *goquery.Selection) {
					file.SetCellValue(sheet, fmt.Sprintf("A%d", row), strings.TrimSpace(s.Text()))
				})
			})

			s.Find("td[data-label=\"Тип\"]").Each(func(i int, s *goquery.Selection) {
				file.SetCellValue(sheet, fmt.Sprintf("B%d", row), strings.TrimSpace(s.Text()))
			})
			row++
		})

		if err := file.SaveAs(*listType + "_anime_list.xlsx"); err != nil {
			log.Fatalf("could not save excel file: %v", err)
		}
		log.Println(fmt.Sprintf("Your data in %s_anime_list.xlsx!", *listType))

	default:
		log.Fatalf("Unsupported output format: %v", *outputFormat)
	}

	fmt.Println("Done.")
}
