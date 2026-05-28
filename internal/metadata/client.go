package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/a3tai/library/internal/library"
)

type Client struct {
	http    *http.Client
	limiter *rateLimiter
}

// New returns a metadata client with sane defaults: 8s HTTP timeout and a
// 4 RPS global rate limit on external lookups. The limiter is shared across
// every goroutine using this client, so the parallel enricher workers can't
// collectively exceed the polite cap on Open Library / Google Books.
func New() *Client {
	return &Client{
		http:    &http.Client{Timeout: 8 * time.Second},
		limiter: newRateLimiter(250 * time.Millisecond),
	}
}

// rateLimiter is a minimal token bucket: at most one request every
// `interval`, mutex-guarded so multiple workers serialise through it.
// Bursts are not allowed — strictly paced. Adequate for the modest fan-out
// the enricher uses; if we ever need true bursting, swap in
// golang.org/x/time/rate.
type rateLimiter struct {
	mu       sync.Mutex
	interval time.Duration
	next     time.Time
}

func newRateLimiter(interval time.Duration) *rateLimiter {
	return &rateLimiter{interval: interval}
}

// Wait blocks until the caller is allowed to make its next request, or
// until ctx is cancelled. Returns ctx.Err() in the cancellation case.
func (r *rateLimiter) Wait(ctx context.Context) error {
	r.mu.Lock()
	now := time.Now()
	if r.next.Before(now) {
		r.next = now
	}
	delay := r.next.Sub(now)
	r.next = r.next.Add(r.interval)
	r.mu.Unlock()
	if delay <= 0 {
		return nil
	}
	select {
	case <-time.After(delay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Client) Lookup(ctx context.Context, book library.Book) (library.Book, bool) {
	if found, ok := c.openLibrary(ctx, book); ok {
		return found, true
	}
	if found, ok := c.googleBooks(ctx, book); ok {
		return found, true
	}
	return library.Book{}, false
}

func (c *Client) openLibrary(ctx context.Context, book library.Book) (library.Book, bool) {
	if isbn := firstISBN(book); isbn != "" {
		if found, ok := c.openLibraryISBN(ctx, isbn); ok {
			return found, true
		}
	}
	return c.openLibrarySearch(ctx, book)
}

func (c *Client) openLibraryISBN(ctx context.Context, isbn string) (library.Book, bool) {
	endpoint := "https://openlibrary.org/api/books?" + url.Values{
		"bibkeys": []string{"ISBN:" + isbn},
		"jscmd":   []string{"data"},
		"format":  []string{"json"},
	}.Encode()
	var response map[string]openLibraryDataBook
	if err := c.getJSON(ctx, endpoint, &response); err != nil {
		return library.Book{}, false
	}
	item, ok := response["ISBN:"+isbn]
	if !ok || strings.TrimSpace(item.Title) == "" {
		return library.Book{}, false
	}
	book := item.toBook()
	book.MetadataSource = "openlibrary"
	return book, true
}

func (c *Client) openLibrarySearch(ctx context.Context, book library.Book) (library.Book, bool) {
	values := url.Values{"limit": []string{"1"}}
	if book.Title != "" {
		values.Set("title", book.Title)
	}
	if book.Authors != "" {
		values.Set("author", strings.Split(book.Authors, ",")[0])
	}
	if len(values) <= 1 {
		return library.Book{}, false
	}
	// Open Library's search.json returns a minimal field set by default —
	// notably *without* `subject`. We have to ask for it explicitly, or
	// every book comes back with an empty description and the
	// Categories / By Subject sidebar views stay blank.
	values.Set("fields", "title,author_name,publisher,first_publish_year,isbn,language,cover_i,subject")
	var response openLibrarySearchResponse
	if err := c.getJSON(ctx, "https://openlibrary.org/search.json?"+values.Encode(), &response); err != nil {
		return library.Book{}, false
	}
	if len(response.Docs) == 0 || response.Docs[0].Title == "" {
		return library.Book{}, false
	}
	result := response.Docs[0].toBook()
	result.MetadataSource = "openlibrary"
	return result, true
}

func (c *Client) googleBooks(ctx context.Context, book library.Book) (library.Book, bool) {
	query := ""
	if isbn := firstISBN(book); isbn != "" {
		query = "isbn:" + isbn
	} else if book.Title != "" {
		query = "intitle:" + book.Title
		if book.Authors != "" {
			query += "+inauthor:" + strings.Split(book.Authors, ",")[0]
		}
	}
	if query == "" {
		return library.Book{}, false
	}
	var response googleBooksResponse
	endpoint := "https://www.googleapis.com/books/v1/volumes?" + url.Values{"q": []string{query}, "maxResults": []string{"1"}}.Encode()
	if err := c.getJSON(ctx, endpoint, &response); err != nil {
		return library.Book{}, false
	}
	if len(response.Items) == 0 || response.Items[0].VolumeInfo.Title == "" {
		return library.Book{}, false
	}
	result := response.Items[0].VolumeInfo.toBook()
	result.MetadataSource = "googlebooks"
	return result, true
}

func (c *Client) getJSON(ctx context.Context, endpoint string, target any) error {
	// Wait for the global rate-limit token before opening the connection.
	// Done outside the request-scoped timeout deliberately — sleeping here
	// is the price of being a good citizen, not a sign the server is slow.
	if c.limiter != nil {
		if err := c.limiter.Wait(ctx); err != nil {
			return err
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "a3t-library/1.0.0")
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return fmt.Errorf("metadata request failed: %s", res.Status)
	}
	return json.NewDecoder(res.Body).Decode(target)
}

func firstISBN(book library.Book) string {
	if book.ISBN13 != "" {
		return book.ISBN13
	}
	return book.ISBN10
}

type openLibraryDataBook struct {
	Title         string `json:"title"`
	PublishDate   string `json:"publish_date"`
	NumberOfPages int    `json:"number_of_pages"`
	Authors       []struct {
		Name string `json:"name"`
	} `json:"authors"`
	Publishers []struct {
		Name string `json:"name"`
	} `json:"publishers"`
	Subjects []struct {
		Name string `json:"name"`
	} `json:"subjects"`
	Cover struct {
		Medium string `json:"medium"`
		Large  string `json:"large"`
	} `json:"cover"`
	Identifiers struct {
		ISBN10 []string `json:"isbn_10"`
		ISBN13 []string `json:"isbn_13"`
	} `json:"identifiers"`
}

func (b openLibraryDataBook) toBook() library.Book {
	authors := make([]string, 0, len(b.Authors))
	for _, author := range b.Authors {
		authors = append(authors, author.Name)
	}
	publisher := ""
	if len(b.Publishers) > 0 {
		publisher = b.Publishers[0].Name
	}
	cover := b.Cover.Large
	if cover == "" {
		cover = b.Cover.Medium
	}
	subjects := make([]string, 0, len(b.Subjects))
	for _, subject := range b.Subjects {
		if subject.Name != "" {
			subjects = append(subjects, subject.Name)
		}
		if len(subjects) == 6 {
			break
		}
	}
	return library.Book{
		Title:         b.Title,
		Authors:       strings.Join(authors, ", "),
		Publisher:     publisher,
		PublishedDate: b.PublishDate,
		ISBN10:        firstString(b.Identifiers.ISBN10),
		ISBN13:        firstString(b.Identifiers.ISBN13),
		CoverURL:      cover,
		Subjects:      subjects,
	}
}

type openLibrarySearchResponse struct {
	Docs []openLibraryDoc `json:"docs"`
}

type openLibraryDoc struct {
	Title        string   `json:"title"`
	AuthorName   []string `json:"author_name"`
	Publisher    []string `json:"publisher"`
	FirstPublish int      `json:"first_publish_year"`
	ISBN         []string `json:"isbn"`
	Language     []string `json:"language"`
	CoverID      int      `json:"cover_i"`
	Subject      []string `json:"subject"`
}

func (d openLibraryDoc) toBook() library.Book {
	book := library.Book{
		Title:         d.Title,
		Authors:       strings.Join(d.AuthorName, ", "),
		Publisher:     firstString(d.Publisher),
		PublishedDate: "",
		Language:      firstString(d.Language),
	}
	if d.FirstPublish > 0 {
		book.PublishedDate = fmt.Sprintf("%d", d.FirstPublish)
	}
	for _, isbn := range d.ISBN {
		if len(isbn) == 10 && book.ISBN10 == "" {
			book.ISBN10 = isbn
		}
		if len(isbn) == 13 && book.ISBN13 == "" {
			book.ISBN13 = isbn
		}
	}
	if d.CoverID > 0 {
		book.CoverURL = fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-L.jpg", d.CoverID)
	}
	if len(d.Subject) > 0 {
		limit := len(d.Subject)
		if limit > 6 {
			limit = 6
		}
		book.Subjects = append([]string(nil), d.Subject[:limit]...)
	}
	return book
}

type googleBooksResponse struct {
	Items []struct {
		VolumeInfo googleVolumeInfo `json:"volumeInfo"`
	} `json:"items"`
}

type googleVolumeInfo struct {
	Title         string   `json:"title"`
	Authors       []string `json:"authors"`
	Publisher     string   `json:"publisher"`
	PublishedDate string   `json:"publishedDate"`
	Description   string   `json:"description"`
	Language      string   `json:"language"`
	Categories    []string `json:"categories"`
	ImageLinks    struct {
		Thumbnail string `json:"thumbnail"`
	} `json:"imageLinks"`
	IndustryIdentifiers []struct {
		Type       string `json:"type"`
		Identifier string `json:"identifier"`
	} `json:"industryIdentifiers"`
}

func (v googleVolumeInfo) toBook() library.Book {
	book := library.Book{
		Title:         v.Title,
		Authors:       strings.Join(v.Authors, ", "),
		Publisher:     v.Publisher,
		PublishedDate: v.PublishedDate,
		Description:   v.Description,
		Language:      v.Language,
		CoverURL:      strings.Replace(v.ImageLinks.Thumbnail, "http://", "https://", 1),
	}
	for _, id := range v.IndustryIdentifiers {
		switch id.Type {
		case "ISBN_10":
			book.ISBN10 = id.Identifier
		case "ISBN_13":
			book.ISBN13 = id.Identifier
		}
	}
	// Google Books returns BISAC-style "Computers / Programming Languages":
	// the slash separates a top-level bucket from finer-grained leaves. Top
	// bucket → category; remaining segments → subjects.
	for _, raw := range v.Categories {
		segs := strings.Split(raw, "/")
		for i, seg := range segs {
			s := strings.TrimSpace(seg)
			if s == "" {
				continue
			}
			if i == 0 {
				book.Categories = append(book.Categories, s)
			} else {
				book.Subjects = append(book.Subjects, s)
			}
		}
	}
	return book
}

func firstString(values []string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
