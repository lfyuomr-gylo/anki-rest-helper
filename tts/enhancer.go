package tts

import (
	"anki-rest-enhancer/enhancerconf"
	"bytes"
	"encoding/xml"
	"github.com/joomcode/errorx"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
)

type TextToSpeechResults struct {
	// Error is non-nil if the whole speech generation failed (e.g. due to malformed configuration)
	Error        error
	TextToSpeech map[string]TextToSpeechResult
}

type TextToSpeechResult struct {
	Error     error
	AudioData []byte
}

func TextToSpeech(conf enhancerconf.Azure, texts map[string]struct{}) TextToSpeechResults {
	client := &http.Client{Timeout: conf.RequestTimeout}

	results := TextToSpeechResults{TextToSpeech: make(map[string]TextToSpeechResult, len(texts))}
	i := 0
	for text := range texts {
		i++ // make i equal to 1 on the first iteration
		log.Printf("Speech synthesis [%d / %d]: call text-to-speech for text %s", i, len(texts), text)

		audio, err := doTextToSpeech(client, apiCallArgs{
			EndpointURL: conf.TTSEndpoint,
			APIKey:      conf.APIKey,
			Language:    conf.Language,
			Voice:       conf.Voice,
			Text:        text,
		}, conf.LogRequests)
		if err != nil {
			results.TextToSpeech[text] = TextToSpeechResult{Error: err}
			continue
		}
		results.TextToSpeech[text] = TextToSpeechResult{AudioData: audio}
	}
	return results
}

type apiCallArgs struct {
	EndpointURL                   *url.URL
	APIKey, Language, Voice, Text string
}

func doTextToSpeech(client *http.Client, args apiCallArgs, logRequest bool) ([]byte, error) {
	req, err := makeTextToSpeechRequest(args)
	if err != nil {
		return nil, err
	}
	if logRequest {
		logReq(req)
	}
	resp, err := client.Do(req)
	if err != nil {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return nil, errorx.TimeoutElapsed.Wrap(err, "Text-to-speech API request timed out")
		}
		return nil, errorx.ExternalError.Wrap(err, "Azure API request failed")
	}
	if logRequest {
		if err := logResp(resp); err != nil {
			return nil, err
		}
	}
	defer func() { _ = resp.Body.Close() }()

	audio, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errorx.ExternalError.Wrap(err, "failed to read Azure response body")
	}
	return audio, nil
}

func makeTextToSpeechRequest(args apiCallArgs) (*http.Request, error) {
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
		Lang:    args.Language,
		XMLNS:   "http://www.w3.org/2001/10/synthesis",
		Voice: voice{
			Name: args.Voice,
			Text: args.Text,
		},
	}
	body, err := xml.Marshal(bodyStruct)
	if err != nil {
		return nil, errorx.IllegalState.Wrap(err, "Failed to construct TTS request body")
	}
	req := &http.Request{
		Method: http.MethodPost,
		URL:    args.EndpointURL,
		Header: http.Header{
			"Ocp-Apim-Subscription-Key": []string{args.APIKey},
			"Content-Type":              []string{"application/ssml+xml"},
			"X-Microsoft-OutputFormat":  []string{"audio-16khz-128kbitrate-mono-mp3"},
		},
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),
	}

	return req, nil
}

func logReq(req *http.Request) {
	log.Println("Calling Azure API...")
	body, _ := io.ReadAll(req.Body)
	_ = req.Body.Close()
	req.Body = io.NopCloser(bytes.NewReader(body))
	_ = req.Write(log.Writer())
	req.Body = io.NopCloser(bytes.NewReader(body))
}

func logResp(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errorx.ExternalError.Wrap(err, "failed to read Azure response body")
	}
	_ = resp.Body.Close()

	resp.Body = io.NopCloser(bytes.NewReader(body))
	log.Println("Got response from Azure API:")
	_ = resp.Write(log.Writer())
	resp.Body = io.NopCloser(bytes.NewReader(body))
	return nil
}
