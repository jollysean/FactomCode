package main

import (
	"net/http"
	"net/url"
	"strings"
	ncdata "NotaryChain/data"
	ncrest "NotaryChain/rest"
	"strconv"
	"fmt"
)

func parse(r *http.Request) (path []string, method string, accept string, form url.Values, err *ncrest.Error) {
	url := strings.TrimSpace(r.URL.Path)
	path = strings.Split(url, "/")
	
	pathlen := len(path)
	lastpath := path[pathlen - 1:pathlen]
	bits := strings.Split(lastpath[0], ".")
	bitslen := len(bits)
	
	if len(bits) > 1 {
		lastpath[0], bits = strings.Join(bits[:bitslen - 1], "."), bits[bitslen - 1:bitslen]
	} else {
		bits = make([]string, 0)
	}
	
	if len(path) > 0 && len(path[0]) == 0 {
		path = path[1:]
	}
	
	if len(path) > 0 && len(path[len(path) - 1]) == 0 {
		path = path[:len(path) - 1]
	}
	
	method = r.Method
	
a:	for _, accept = range r.Header["Accept"] {
		for _, accept = range strings.Split(accept, ",") {
			accept, err = parseAccept(accept, bits)
			if err == nil {
				break a
			}
		}
	}
	
	if err != nil {
		return
	}
	
	if accept == "" {
		accept = "json"
	}
	
	e := r.ParseForm()
	if e != nil {
		err = ncrest.CreateError(ncrest.ErrorBadPOSTData, e.Error())
		return
	}
	
	form = r.Form
	
	return
}

func parseAccept(accept string, ext []string) (string, *ncrest.Error) {
	switch accept {
	case "text/plain":
		if len(ext) == 1 && ext[0] != "txt" {
			return ext[0], nil
		}
		return "text", nil
		
	case "application/json", "*/*":
		return "json", nil
		
	case "application/xml", "text/xml":
		return "xml", nil
		
	case "text/html":
		return "html", nil
	}
	
	return "", ncrest.CreateError(ncrest.ErrorNotAcceptable, fmt.Sprintf("The specified resource cannot be returned as %s", accept))
}

func find(path []string) (interface{}, *ncrest.Error) {
	if len(path) == 0 {
		return nil, ncrest.CreateError(ncrest.ErrorMissingVersionSpec, "")
	}
	
	ver, path := path[0], path[1:] // capture version spec
	
	if !strings.HasPrefix(ver, "v") {
		return nil, ncrest.CreateError(ncrest.ErrorMalformedVersionSpec, fmt.Sprintf(`The version specifier "%s" is malformed`, ver))
	}
	
	ver = strings.TrimPrefix(ver, "v")
	
	if ver == "1" {
		return findV1("/v1", path)
	}
	
	return nil, ncrest.CreateError(ncrest.ErrorBadVersionSpec, fmt.Sprintf(`The version specifier "v%s" does not refer to a supported version`, ver))
}

func findV1(context string, path []string) (interface{}, *ncrest.Error) {
	if len(path) == 0 {
		return nil, ncrest.CreateError(ncrest.ErrorEmptyRequest, "")
	}
	
	root, path := path[0], path[1:] // capture root spec
	
	if strings.ToLower(root) != "blocks" {
		return nil, ncrest.CreateError(ncrest.ErrorBadElementSpec, fmt.Sprintf(`The element specifier "%s" is not valid in the context "%s"`, root, context))
	}
	
	return findV1InBlocks(context + "/" + root, path, blocks)
}

func findV1InBlocks(context string, path []string, blocks []*ncdata.Block) (interface{}, *ncrest.Error) {
	if len(path) == 0 {
		return blocks, nil
	}
	
	// capture root spec
	sid := path[0]
	iid, err := strconv.Atoi(sid)
	id := uint64(iid)
	path = path[1:]
	
	if err != nil {
		return nil, ncrest.CreateError(ncrest.ErrorBadIdentifier, fmt.Sprintf(`The identifier "%s" is malformed: %s`, sid, err.Error()))
	}
	
	if len(blocks) == 0 {
		return nil, ncrest.CreateError(ncrest.ErrorBlockNotFound, fmt.Sprintf(`The no blocks can be found in the context "%s"`, sid, context))
	}
	
	idOffset := blocks[0].BlockID
	
	if id < idOffset {
		return nil, ncrest.CreateError(ncrest.ErrorBlockNotFound, fmt.Sprintf(`The block identified by "%s" cannot be found in the context "%s"`, sid, context))
	}
	
	id = id - idOffset
	
	if len(blocks) <= int(id) {
		return nil, ncrest.CreateError(ncrest.ErrorBlockNotFound, fmt.Sprintf(`The block identified by "%s" cannot be found in the context "%s"`, sid, context))
	}
	
	return findV1InBlock(context + "/" + sid, path, blocks[id])
}

func findV1InBlock(context string, path []string, block *ncdata.Block) (interface{}, *ncrest.Error) {
	if len(path) == 0 {
		return block, nil
	}
	
	root, path := path[0], path[1:] // capture root spec
	
	if strings.ToLower(root) != "entries" {
		return nil, nil // bad root spec
	}
	
	return findV1InEntries(context + "/" + root, path, block.Entries)
}

func findV1InEntries(context string, path []string, entries []*ncdata.PlainEntry) (interface{}, *ncrest.Error) {
	if len(path) == 0 {
		return entries, nil
	}
	
	// capture root spec
	sid := path[0]
	id, err := strconv.Atoi(path[0])
	path = path[1:]
	
	if err != nil {
		return nil, ncrest.CreateError(ncrest.ErrorBadIdentifier, fmt.Sprintf(`The identifier "%s" is malformed%s`, sid, err.Error()))
	}
	
	if len(entries) <= id {
		return nil, ncrest.CreateError(ncrest.ErrorEntryNotFound, fmt.Sprintf(`The entry identified by "%s" cannot be found in the context "%s"`, sid, context))
	}
	
	return entries[id], nil
}