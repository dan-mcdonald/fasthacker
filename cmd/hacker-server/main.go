package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"
)

type fastHacker struct {
	staticHandler http.Handler
	indexTmpl     *template.Template
	itemTmpl      *template.Template
	dl            DataLoader
}

func ago(t Time) string {
	return time.Since(t.Time).Round(time.Second).String()
}

type StoryListPage struct {
	RankOffset int
	Stories    []Story
}

type TraversedComment struct {
	Comment Comment
	Level   int
}

type StoryPage struct {
	Story       *Story
	CommentTree []TraversedComment
}

func GetCommentTree(story Story, dl *DataLoader) ([]TraversedComment, error) {
	var commentTraversal []TraversedComment
	var traverse func(CommentID, int) error
	traverse = func(commentId CommentID, level int) error {
		comment, err := (*dl).GetComment(commentId)
		if err != nil {
			return err
		}
		commentTraversal = append(commentTraversal, TraversedComment{
			Comment: comment,
			Level:   level,
		})
		for _, kidId := range comment.Kids {
			err = traverse(kidId, level+1)
			if err != nil {
				return err
			}
		}
		return nil
	}
	for _, commentId := range story.Kids {
		err := traverse(commentId, 0)
		if err != nil {
			return nil, err
		}
	}
	return commentTraversal, nil
}

func (srv *fastHacker) handleItem(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	storyId := StoryID(0)
	fmt.Sscanf(id, "%d", &storyId)
	story, err := srv.dl.GetStory(storyId)
	if err != nil {
		log.Printf("handleItem GetStory(%d): %s", storyId, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	commentTree, err := GetCommentTree(story, &srv.dl)
	if err != nil {
		log.Printf("handleItem GetCommentTree(%d): %s", storyId, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	data := StoryPage{
		Story:       &story,
		CommentTree: commentTree,
	}
	err = srv.itemTmpl.Execute(w, data)
	if err != nil {
		log.Printf("handleItem template execute(): %s", err)
	}
}

func (srv *fastHacker) handleDefault(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		srv.staticHandler.ServeHTTP(w, r)
		return
	}

	topStories, err := srv.dl.GetTopStories()
	if err != nil {
		log.Printf("handleIndex GetNewsPosts(): %s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	var stories []Story
	for idx, storyId := range topStories {
		if idx > 30 {
			break
		}
		story, err := srv.dl.GetStory(storyId)
		if err != nil {
			log.Printf("handleIndex GetStory(%d): %s", storyId, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		stories = append(stories, story)
	}
	data := StoryListPage{
		RankOffset: 1,
		Stories:    stories,
	}
	err = srv.indexTmpl.Execute(w, data)
	if err != nil {
		log.Printf("handleIndex template execute(): %s", err)
	}
}

func rfc3339(t Time) string {
	return t.UTC().Format(time.RFC3339)
}

func main() {
	fmt.Println("fasthacker starting")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv := &http.Server{
		Addr: "localhost:8080",
	}

	funcMap := template.FuncMap{
		"add":      func(a, b int) int { return a + b },
		"multiply": func(a, b int) int { return a * b },
		"ago":      ago,
		"rfc3339":  rfc3339,
	}
	indexTmpl := template.New("index.html")
	indexTmpl.Funcs(funcMap)
	indexTmpl, err := indexTmpl.ParseFiles("templates/index.html")
	if err != nil {
		log.Fatalf("ParseFiles(): %s", err)
	}

	itemTmpl := template.New("item.html")
	itemTmpl.Funcs(funcMap)
	itemTmpl, err = itemTmpl.ParseFiles("templates/item.html")
	if err != nil {
		log.Fatalf("ParseFiles(): %s", err)
	}

	fastHacker := &fastHacker{
		indexTmpl:     indexTmpl,
		itemTmpl:      itemTmpl,
		staticHandler: http.FileServer(http.Dir("static")),
		dl:            NewLoader(ctx),
	}
	http.HandleFunc("/", fastHacker.handleDefault)
	http.HandleFunc("/item", fastHacker.handleItem)

	log.Println("Starting server on http://localhost:8080")
	err = srv.ListenAndServe()
	if err != http.ErrServerClosed {
		log.Fatalf("ListenAndServe(): %s", err)
	}
	log.Println("Server stopped")
}
