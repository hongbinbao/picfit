package application

import (
	"github.com/thoas/kvstores"
	"github.com/thoas/muxer"
	"github.com/thoas/picfit/extractors"
	"github.com/thoas/picfit/hash"
	"github.com/thoas/picfit/image"
	"net/http"
	"net/url"
)

func NotFound(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "404 not found", http.StatusNotFound)
}

func NotFoundHandler() http.Handler {
	return http.HandlerFunc(NotFound)
}

type Request struct {
	*muxer.Request
	Operation  *image.Operation
	Connection kvstores.KVStoreConnection
	Key        string
	URL        *url.URL
	Filepath   string
}

type Handler func(muxer.Response, *Request)

func (h Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	con := App.KVStore.Connection()
	defer con.Close()

	request := muxer.NewRequest(req)

	for k, v := range request.Params {
		request.QueryString[k] = v
	}

	operation, err := extractors.Operation(request)

	res := muxer.NewResponse(w)

	if err != nil {
		res.BadRequest()
		return
	}

	url, err := extractors.URL(request)

	filepath := request.QueryString["path"]

	if err != nil && filepath == "" {
		res.BadRequest()
		return
	}

	qs := request.QueryString

	delete(qs, "sig")

	key := hash.Tokey(hash.Serialize(qs))

	reqURL := *req.URL

	if !App.IsValidSign(reqURL.RawQuery) {
		res.BadRequest()
		return
	}

	h(res, &Request{request, operation, con, key, url, filepath})
}

var ImageHandler Handler = func(res muxer.Response, req *Request) {
	file, err := App.ImageFileFromRequest(req, true, true)

	panicIf(err)

	content, err := file.ToBytes()

	panicIf(err)

	res.SetHeaders(file.Header, true)
	res.ResponseWriter.Write(content)
}

var GetHandler Handler = func(res muxer.Response, req *Request) {
	file, err := App.ImageFileFromRequest(req, false, false)

	panicIf(err)

	content, err := App.ToJSON(file)

	panicIf(err)

	res.ContentType("application/json")
	res.ResponseWriter.Write(content)
}

var RedirectHandler Handler = func(res muxer.Response, req *Request) {
	file, err := App.ImageFileFromRequest(req, false, false)

	panicIf(err)

	res.PermanentRedirect(file.GetURL())
}
