// Package fusionbrain - The Fusion Brain API. Documentation: https://fusionbrain.ai/docs/en/doc/api-dokumentaciya/
package fusionbrain

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/textproto"
	"strconv"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fastjson"
)

// The Fusion Brain API urls
const (
	uriGetModels       = "https://api-key.fusionbrain.ai/key/api/v1/models"
	uriTextToImage     = "https://api-key.fusionbrain.ai/key/api/v1/text2image/run"
	uriGetAvailability = "https://api-key.fusionbrain.ai/key/api/v1/text2image/availability"
	uriGetCheckStatus  = "https://api-key.fusionbrain.ai/key/api/v1/text2image/status/"
	uriGetStyles       = "https://cdn.fusionbrain.ai/static/styles/key"
)

// The Fusion Brain statuses
const (
	// StatusInitial - the request has been received, is in the queue for processing
	StatusInitial = "INITIAL"
	// StatusProcessing - the request is being processed
	StatusProcessing = "PROCESSING"
	// StatusDone - task completed
	StatusDone = "DONE"
	// StatusFail - the task could not be completed
	StatusFail = "FAIL"
	// StatusDisabledByQueue - service internal error
	StatusDisabledByQueue = "DISABLED_BY_QUEUE"
)

const (
	headerXKey    = "X-Key"
	headerXSecret = "X-Secret"

	formModelID = "model_id"
	formParams  = "params"
)

var (
	errStatusNotInitial        = errors.New("status not INITIAL")
	errStatusIsDisabledByQueue = errors.New("status is DISABLED_BY_QUEUE")
	errEmptyUUID               = errors.New("empty UUID")
)

type Style struct {
	Name    string `json:"name"`
	Title   string `json:"title"`
	TitleEn string `json:"titleEn"`
	Image   string `json:"image"`
}

type GenerationStatus struct {
	UUID             string   `json:"uuid"`
	Status           string   `json:"status"`
	Images           []string `json:"images"`
	ErrorDescription string   `json:"errorDescription"`
	Censored         string   `json:"censored"`
}

type Model struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Type    string `json:"type"`
}

type RequestBody struct {
	Prompt, NegativePrompt, Style string
	Width, Height                 int
}

func (r *RequestBody) serialize() []byte {
	var b bytes.Buffer
	b.WriteString(`{"type":"GENERATE","style":"`)
	b.WriteString(r.Style)
	b.WriteString(`","width":`)
	b.WriteString(strconv.Itoa(r.Width))
	b.WriteString(`,"height":`)
	b.WriteString(strconv.Itoa(r.Height))
	b.WriteString(`,"num_images":1,"negativePromptUnclip":"`)
	b.WriteString(r.NegativePrompt)
	b.WriteString(`","generateParams":{"query":"`)
	b.WriteString(r.Prompt)
	b.WriteString(`"}}`)
	return b.Bytes()
}

type FusionBrain struct {
	client          *fasthttp.Client
	key, secretKey  string
	errsByRespCodes map[int]error
}

// NewFusionBrain - the Fusion Brain API is a new section of the platform that allows platform users to access artificial intelligence models via the API.
// One of the first models that is available via the API is the Kandinsky model.
func NewFusionBrain(client *fasthttp.Client, key, secretKey string) *FusionBrain {
	return &FusionBrain{
		client:    client,
		key:       "Key " + key,
		secretKey: "Secret " + secretKey,
		errsByRespCodes: map[int]error{
			fasthttp.StatusUnauthorized:         errors.New("401 authorisation error"),
			fasthttp.StatusNotFound:             errors.New("404 not found"),
			fasthttp.StatusBadRequest:           errors.New("400 invalid request parameters or the text description is too long"),
			fasthttp.StatusInternalServerError:  errors.New("500 server error when executing the request"),
			fasthttp.StatusUnsupportedMediaType: errors.New("415 the content format is not supported by the server"),
		},
	}
}

// CheckAvailable - if there is a heavy load or technical work, the service may be temporarily unavailable to accept new tasks.
// You can check the current status in advance by using a GET request to URL /key/api/v1/text2image/availability.
// During unavailability, tasks will not be accepted and in response to a request to the model,
// instead of the uuid of your task, the current status of the service will be returned.
func (f *FusionBrain) CheckAvailable(ctx context.Context, modelID int) error {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	req.SetRequestURI(uriGetAvailability)
	req.Header.SetMethod(fasthttp.MethodGet)
	req.Header.Set(headerXKey, f.key)
	req.Header.Set(headerXSecret, f.secretKey)
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	modelIDPartHeader := textproto.MIMEHeader{}
	modelIDPartHeader.Add("Content-Disposition", `form-data; name="`+formModelID+`"`)
	modelIDPart, err := w.CreatePart(modelIDPartHeader)
	if err != nil {
		return err
	}
	if _, err = modelIDPart.Write([]byte(strconv.Itoa(modelID))); err != nil {
		return err
	}
	if err = w.Close(); err != nil {
		return err
	}
	req.Header.SetContentType(w.FormDataContentType())
	req.SetBody(body.Bytes())
	errChan := make(chan error, 1)
	go func() {
		if err = f.client.Do(req, resp); err != nil {
			errChan <- err
			return
		}
		if err, ok := f.errsByRespCodes[resp.StatusCode()]; ok {
			errChan <- err
			return
		}
		var p fastjson.Parser
		v, err := p.ParseBytes(resp.Body())
		if err != nil {
			errChan <- err
			return
		}
		if string(v.GetStringBytes("model_status")) == StatusDisabledByQueue {
			errChan <- errStatusIsDisabledByQueue
			return
		}
		errChan <- nil
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err = <-errChan:
		return err
	}
}

// GetModels - list of available models. The appeal takes place at the URL: https://api-key.fusionbrain.ai/key/api/v1/models.
func (f *FusionBrain) GetModels(ctx context.Context) ([]Model, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	req.SetRequestURI(uriGetModels)
	req.Header.SetMethod(fasthttp.MethodGet)
	req.Header.Set(headerXKey, f.key)
	req.Header.Set(headerXSecret, f.secretKey)
	modelsChan := make(chan []Model, 1)
	errChan := make(chan error, 1)
	go func() {
		if err := f.client.Do(req, resp); err != nil {
			errChan <- err
			return
		}
		if err, ok := f.errsByRespCodes[resp.StatusCode()]; ok {
			errChan <- err
			return
		}
		var p fastjson.Parser
		val, err := p.ParseBytes(resp.Body())
		if err != nil {
			errChan <- err
			return
		}
		var models []Model
		arr, _ := val.Array()
		for _, v := range arr {
			model := Model{
				ID:      v.GetInt("id"),
				Name:    string(v.GetStringBytes("name")),
				Version: string(v.GetStringBytes("version")),
				Type:    string(v.GetStringBytes("type")),
			}
			models = append(models, model)
		}
		modelsChan <- models
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case models := <-modelsChan:
		return models, nil
	case err := <-errChan:
		return nil, err
	}
}

// GetStyles - getting the current list of styles.
func (f *FusionBrain) GetStyles(ctx context.Context) ([]Style, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	req.SetRequestURI(uriGetStyles)
	req.Header.SetMethod(fasthttp.MethodGet)
	stylesChan := make(chan []Style, 1)
	errChan := make(chan error, 1)
	go func() {
		if err := f.client.Do(req, resp); err != nil {
			errChan <- err
			return
		}
		if err, ok := f.errsByRespCodes[resp.StatusCode()]; ok {
			errChan <- err
			return
		}
		var p fastjson.Parser
		val, err := p.ParseBytes(resp.Body())
		if err != nil {
			errChan <- err
			return
		}
		var styles []Style
		arr, _ := val.Array()
		for _, v := range arr {
			style := Style{
				Name:    string(v.GetStringBytes("name")),
				Title:   string(v.GetStringBytes("title")),
				TitleEn: string(v.GetStringBytes("titleEn")),
				Image:   string(v.GetStringBytes("image")),
			}
			styles = append(styles, style)
		}
		stylesChan <- styles
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case styles := <-stylesChan:
		return styles, nil
	case err := <-errChan:
		return nil, err
	}
}

// TextToImage - the generate mode takes a text description of an image as input and generates an image corresponding to it.
// To call this method, you need to send a POST request to the URL /key/api/v1/text2image/run.
func (f *FusionBrain) TextToImage(ctx context.Context, reqBody RequestBody, modelID int) (string, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	req.SetRequestURI(uriTextToImage)
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.Set(headerXKey, f.key)
	req.Header.Set(headerXSecret, f.secretKey)
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	paramsPartHeader := textproto.MIMEHeader{}
	paramsPartHeader.Add("Content-Disposition", `form-data; name="`+formParams+`"`)
	paramsPartHeader.Add("Content-Type", "application/json")
	paramsPart, err := w.CreatePart(paramsPartHeader)
	if err != nil {
		return "", err
	}
	if _, err = paramsPart.Write(reqBody.serialize()); err != nil {
		return "", err
	}
	modelIDPartHeader := textproto.MIMEHeader{}
	modelIDPartHeader.Add("Content-Disposition", `form-data; name="`+formModelID+`"`)
	modelIDPart, err := w.CreatePart(modelIDPartHeader)
	if err != nil {
		return "", err
	}
	if _, err = modelIDPart.Write([]byte(strconv.Itoa(modelID))); err != nil {
		return "", err
	}
	if err = w.Close(); err != nil {
		return "", err
	}
	req.Header.SetContentType(w.FormDataContentType())
	req.SetBody(body.Bytes())
	uuidChan := make(chan string, 1)
	errChan := make(chan error, 1)
	go func() {
		if err = f.client.Do(req, resp); err != nil {
			errChan <- err
			return
		}
		var ok bool
		if err, ok = f.errsByRespCodes[resp.StatusCode()]; ok {
			errChan <- err
			return
		}
		var (
			p fastjson.Parser
			v *fastjson.Value
		)
		v, err = p.ParseBytes(resp.Body())
		if err != nil {
			errChan <- err
			return
		}
		if string(v.GetStringBytes("status")) != StatusInitial {
			errChan <- errStatusNotInitial
			return
		}
		uuid := string(v.GetStringBytes("uuid"))
		if uuid == "" {
			errChan <- errEmptyUUID
			return
		}
		uuidChan <- uuid
	}()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case uuid := <-uuidChan:
		return uuid, nil
	case err = <-errChan:
		return "", err
	}
}

// CheckStatus - The check_status request allows you to check the status of image generation.
// To call this method, you need to send a GET request to the URL запрос на URL /key/api/v1/text2image/status/{uuid},
// where uid is the task ID received when calling the request to the model.
func (f *FusionBrain) CheckStatus(ctx context.Context, uuid string) (GenerationStatus, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	req.SetRequestURI(uriGetCheckStatus + uuid)
	req.Header.SetMethod(fasthttp.MethodGet)
	req.Header.Set(headerXKey, f.key)
	req.Header.Set(headerXSecret, f.secretKey)
	respChan := make(chan GenerationStatus, 1)
	errChan := make(chan error, 1)
	go func() {
		if err := f.client.Do(req, resp); err != nil {
			errChan <- err
			return
		}
		if err, ok := f.errsByRespCodes[resp.StatusCode()]; ok {
			errChan <- err
			return
		}
		var p fastjson.Parser
		v, err := p.ParseBytes(resp.Body())
		if err != nil {
			errChan <- err
			return
		}
		var images []string
		for _, imagesVal := range v.GetArray("images") {
			if imagesVal == nil {
				continue
			}
			images = append(images, imagesVal.String())
		}
		respChan <- GenerationStatus{
			UUID:             string(v.GetStringBytes("uuid")),
			Status:           string(v.GetStringBytes("status")),
			Images:           images,
			ErrorDescription: string(v.GetStringBytes("errorDescription")),
			Censored:         string(v.GetStringBytes("censored")),
		}
	}()
	select {
	case <-ctx.Done():
		return GenerationStatus{}, ctx.Err()
	case responseCheckStatus := <-respChan:
		return responseCheckStatus, nil
	case err := <-errChan:
		return GenerationStatus{}, err
	}
}
