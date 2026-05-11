package cbr

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/beevik/etree"
)

type KeyRate struct {
	Rate float64
	Date string
}

type Client struct {
	endpoint   string
	httpClient *http.Client
}

func NewClient(endpoint string) *Client {
	return &Client{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) GetKeyRate(ctx context.Context) (*KeyRate, error) {
	soapRequest := buildSOAPRequest()

	rawBody, err := c.sendRequest(ctx, soapRequest)
	if err != nil {
		return nil, err
	}

	keyRate, err := parseXMLResponse(rawBody)
	if err != nil {
		return nil, err
	}

	return keyRate, nil
}

func buildSOAPRequest() string {
	fromDate := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	toDate := time.Now().Format("2006-01-02")

	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<soap12:Envelope xmlns:soap12="http://www.w3.org/2003/05/soap-envelope">
	<soap12:Body>
		<KeyRate xmlns="http://web.cbr.ru/">
			<fromDate>%s</fromDate>
			<ToDate>%s</ToDate>
		</KeyRate>
	</soap12:Body>
</soap12:Envelope>`, fromDate, toDate)
}

func (c *Client) sendRequest(ctx context.Context, soapRequest string) ([]byte, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.endpoint,
		bytes.NewBuffer([]byte(soapRequest)),
	)
	if err != nil {
		return nil, fmt.Errorf("create cbr request: %w", err)
	}

	req.Header.Set("Content-Type", "application/soap+xml; charset=utf-8")
	req.Header.Set("SOAPAction", "http://web.cbr.ru/KeyRate")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send cbr request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read cbr response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("cbr response status: %d", resp.StatusCode)
	}

	return rawBody, nil
}

func parseXMLResponse(rawBody []byte) (*KeyRate, error) {
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(rawBody); err != nil {
		return nil, fmt.Errorf("parse cbr xml: %w", err)
	}

	root := doc.Root()
	if root == nil {
		return nil, errors.New("empty cbr xml")
	}

	krElements := findElementsByTag(root, "KR")
	if len(krElements) == 0 {
		return nil, errors.New("key rate data not found")
	}

	var selected *parsedKeyRate

	for _, element := range krElements {
		parsed, err := parseKeyRateElement(element)
		if err != nil {
			continue
		}

		if selected == nil {
			selected = parsed
			continue
		}

		if parsed.hasDate && (!selected.hasDate || parsed.date.After(selected.date)) {
			selected = parsed
		}
	}

	if selected == nil {
		return nil, errors.New("valid key rate data not found")
	}

	return &KeyRate{
		Rate: selected.rate,
		Date: selected.dateText,
	}, nil
}

type parsedKeyRate struct {
	rate     float64
	date     time.Time
	dateText string
	hasDate  bool
}

func parseKeyRateElement(element *etree.Element) (*parsedKeyRate, error) {
	rateText, ok := findChildTextByTag(element, "Rate")
	if !ok {
		return nil, errors.New("rate tag not found")
	}

	rate, err := parseRate(rateText)
	if err != nil {
		return nil, err
	}

	dateText, ok := findChildTextByTag(element, "DT")
	if !ok {
		return &parsedKeyRate{
			rate: rate,
		}, nil
	}

	date, hasDate := parseOptionalDate(dateText)
	if !hasDate {
		return &parsedKeyRate{
			rate:     rate,
			dateText: strings.TrimSpace(dateText),
		}, nil
	}

	return &parsedKeyRate{
		rate:     rate,
		date:     date,
		dateText: date.Format("2006-01-02"),
		hasDate:  true,
	}, nil
}

func parseOptionalDate(dateText string) (time.Time, bool) {
	date, err := parseDate(dateText)
	if err != nil {
		return time.Time{}, false
	}

	return date, true
}

func parseRate(rateText string) (float64, error) {
	rateText = strings.TrimSpace(rateText)
	rateText = strings.ReplaceAll(rateText, ",", ".")

	rate, err := strconv.ParseFloat(rateText, 64)
	if err != nil {
		return 0, fmt.Errorf("parse key rate: %w", err)
	}

	return rate, nil
}

func parseDate(dateText string) (time.Time, error) {
	dateText = strings.TrimSpace(dateText)

	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		date, err := time.Parse(format, dateText)
		if err == nil {
			return date, nil
		}
	}

	return time.Time{}, fmt.Errorf("parse key rate date: %s", dateText)
}

func findElementsByTag(root *etree.Element, tag string) []*etree.Element {
	result := make([]*etree.Element, 0)

	var walk func(element *etree.Element)
	walk = func(element *etree.Element) {
		if strings.EqualFold(element.Tag, tag) {
			result = append(result, element)
		}

		for _, child := range element.ChildElements() {
			walk(child)
		}
	}

	walk(root)

	return result
}

func findChildTextByTag(element *etree.Element, tag string) (string, bool) {
	for _, child := range element.ChildElements() {
		if strings.EqualFold(child.Tag, tag) {
			return child.Text(), true
		}
	}

	return "", false
}
