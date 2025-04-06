package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type Course struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func getCanvasCourses(baseURL, accessToken string) ([]Course, error) {
	var courses []Course
	url := fmt.Sprintf("%s/api/v1/courses?per_page=100", baseURL)

	client := &http.Client{}

	for url != "" {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("API error: %s\n%s", resp.Status, string(body))
		}

		var pageCourses []Course
		if err := json.NewDecoder(resp.Body).Decode(&pageCourses); err != nil {
			return nil, err
		}

		courses = append(courses, pageCourses...)

		linkHeader := resp.Header.Get("Link")
		url = getNextPageURL(linkHeader)
	}

	var spring2025Courses []Course
	for _, course := range courses {
		if strings.Contains(course.Name, "Spring 2025") {
			spring2025Courses = append(spring2025Courses, course)
		}
	}

	return spring2025Courses, nil
}

func getNextPageURL(linkHeader string) string {
	var nextURL string
	links := parseLinkHeader(linkHeader)
	if url, ok := links["next"]; ok {
		nextURL = url
	}
	return nextURL
}

func parseLinkHeader(header string) map[string]string {
	links := make(map[string]string)
	var rel, url string
	for _, part := range splitAndTrim(header, ",") {
		segments := splitAndTrim(part, ";")
		if len(segments) >= 2 {
			url = segments[0]
			url = url[1 : len(url)-1]
			for _, s := range segments[1:] {
				if len(s) > 5 && s[:4] == `rel=` {
					rel = s[5 : len(s)-1]
					links[rel] = url
				}
			}
		}
	}
	return links
}

func splitAndTrim(s, sep string) []string {
	raw := make([]string, 0)
	for _, part := range splitNonEmpty(s, sep) {
		raw = append(raw, trim(part))
	}
	return raw
}

func splitNonEmpty(s, sep string) []string {
	parts := make([]string, 0)
	for _, p := range split(s, sep) {
		if trim(p) != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

func split(s, sep string) []string {
	return []string{
		s[:find(s, sep)],
		s[find(s, sep)+1:],
	}
}

func find(s, sep string) int {
	for i := range s {
		if len(s)-i >= len(sep) && s[i:i+len(sep)] == sep {
			return i
		}
	}
	return -1
}

func trim(s string) string {
	return string([]byte(s)[trimLeft(s):trimRight(s)])
}

func trimLeft(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] != ' ' && s[i] != '\t' {
			return i
		}
	}
	return 0
}

func trimRight(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] != ' ' && s[i] != '\t' {
			return i + 1
		}
	}
	return len(s)
}

func CoursesIndex(c *gin.Context) {
	session := sessions.Default(c)
	canvasToken := session.Get("canvas")

	if canvasToken == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "No Canvas token in session"})
		return
	}

	baseURL := "https://usfca.instructure.com"

	courses, err := getCanvasCourses(baseURL, canvasToken.(string))
	if err != nil {
		c.HTML(http.StatusInternalServerError, "courses/index.tpl", gin.H{
			"error":   err.Error(),
			"courses": nil,
		})
		return
	}

	/*
		for _, course := range courses {
			models.CreateCourse(course.Number, course.Name)
		}
	*/

	c.HTML(http.StatusOK, "courses/index.tpl", gin.H{
		"canvas":  canvasToken,
		"courses": courses,
	})
}

func CoursesShow(c *gin.Context) {
	session := sessions.Default(c)
	canvas := session.Get("canvas")
	c.HTML(
		http.StatusOK,
		"canvas/show.tpl",
		gin.H{
			"canvas": canvas,
		},
	)
}
