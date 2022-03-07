package azuretts

import (
	"anki-rest-enhancer/enhancerconf"
	"anki-rest-enhancer/util/httputil"
	"bytes"
	"encoding/xml"
	"github.com/joomcode/errorx"
	"io"
	"log"
	"net"
	"net/http"
)

func NewAPI(conf enhancerconf.Azure) *api {
	client := &http.Client{
		Timeout: conf.RequestTimeout,
	}
	client.Transport = httputil.NewThrottlingTransport(http.DefaultTransport, conf.MinPauseBetweenRequests)
	if conf.LogRequests {
		client.Transport = httputil.NewLoggingRoundTripper(client.Transport)
	}

	return &api{client: client, conf: conf}
}

type api struct {
	client *http.Client
	conf   enhancerconf.Azure
}

func (api api) TextToSpeech(texts map[string]struct{}) map[string]TextToSpeechResult {
	results := make(map[string]TextToSpeechResult, len(texts))
	i := 0
	for text := range texts {
		i++ // make i equal to 1 on the first iteration
		log.Printf("Speech synthesis [%d / %d]: call text-to-speech for text %q", i, len(texts), text)

		var audio []byte
		var err error
		for {
			audio, err = api.doTextToSpeech(text)
			if err != nil && api.conf.RetryOnTooManyRequests && errorx.IsOfType(err, TooManyRequests) {
				// TODO: limit the number of retries.
				log.Println("Got Too Many Requests from Azure. Retry the request...")
				continue
			}
			break
		}
		if err != nil {
			results[text] = TextToSpeechResult{Error: err}
			continue
		}
		results[text] = TextToSpeechResult{AudioMP3: audio}
	}
	return results
}

func (api api) doTextToSpeech(text string) ([]byte, error) {
	req, err := api.makeTextToSpeechRequest(text)
	if err != nil {
		return nil, err
	}
	resp, err := api.client.Do(req)
	if err != nil {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return nil, errorx.TimeoutElapsed.Wrap(err, "Text-to-speech API request timed out")
		}
		return nil, errorx.ExternalError.Wrap(err, "Azure API request failed")
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, TooManyRequests.NewWithNoMessage()
		}
		const maxBodySize = 1000
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
		body := string(bodyBytes)
		if len(body) == maxBodySize {
			body += "..."
		}
		return nil, errorx.ExternalError.New("Azure returned non-200 status code %d with the following body: %s", resp.StatusCode, body)
	}

	audio, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errorx.ExternalError.Wrap(err, "failed to read Azure response body")
	}
	return audio, nil
}

func (api api) makeTextToSpeechRequest(text string) (*http.Request, error) {
	type voice struct {
		Name string `xml:"name,attr"`
		Text string `xml:",chardata"`
	}
	type speak struct {
		XMLName xml.Name `xml:"speak"`

		Version string `xml:"version,attr"`
		Lang    string `xml:"xml:lang,attr"`
		XMLNS   string `xml:"xmlns,attr"`

		Voice voice `xml:"voice"`
	}

	bodyStruct := speak{
		Version: "1.0",
		Lang:    api.conf.Language,
		XMLNS:   "http://www.w3.org/2001/10/synthesis",
		Voice: voice{
			Name: api.conf.Voice,
			Text: text,
		},
	}
	body, err := xml.Marshal(bodyStruct)
	if err != nil {
		return nil, errorx.IllegalState.Wrap(err, "Failed to construct TTS request body")
	}
	req := &http.Request{
		Method: http.MethodPost,
		URL:    api.conf.EndpointURL,
		Header: http.Header{
			"Ocp-Apim-Subscription-Key": []string{api.conf.APIKey},
			"Content-Type":              []string{"application/ssml+xml"},
			"X-Microsoft-OutputFormat":  []string{"audio-24khz-160kbitrate-mono-mp3"},
		},
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),
	}

	return req, nil
}
