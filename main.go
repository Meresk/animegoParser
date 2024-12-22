package main

import (
	"flag"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/playwright-community/playwright-go"
	"log"
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
	flag.Parse()

	if len(*user) == 0 {
		log.Fatal("user is required")
	}
	if !allowedListTypes[*listType] {
		log.Println("list type is invalid, it will be all")
		*listType = "all"
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

	// Находим строки с данными
	doc.Find("table tr").Each(func(i int, s *goquery.Selection) {
		fmt.Println("-----------------------------")
		s.Find("td[class=\"text-left table-100\"]").Each(func(i int, s *goquery.Selection) {
			// Извлекаем текст из <div>
			s.Find("div").Each(func(i int, s *goquery.Selection) {
				fmt.Println(strings.TrimSpace(s.Text()))
			})
			// Извлекаем текст из <a>
			s.Find("a").Each(func(i int, s *goquery.Selection) {
				fmt.Println(strings.TrimSpace(s.Text()))
			})
		})

		s.Find("td[data-label=\"Тип\"]").Each(func(i int, s *goquery.Selection) {
			fmt.Println(strings.TrimSpace(s.Text()))
		})
	})

	fmt.Println("Done.")
}
