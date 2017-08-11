package controller

// this file contains some paging related utility functions

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
)

const (
	pageSizeDefault = 20
	pageSizeMax     = 100
)

func computePagingLimits(offsetParam *string, limitParam *int) (offset int, limit int) {
	if offsetParam == nil {
		offset = 0
	} else {
		offsetValue, err := strconv.Atoi(*offsetParam)
		if err != nil {
			offset = 0
		} else {
			offset = offsetValue
		}
	}
	if offset < 0 {
		offset = 0
	}

	if limitParam == nil {
		limit = pageSizeDefault
	} else {
		limit = *limitParam
	}

	if limit <= 0 {
		limit = pageSizeDefault
	} else if limit > pageSizeMax {
		limit = pageSizeMax
	}
	return offset, limit
}

func setPagingLinks(links *app.PagingLinks, path string, resultLen, offset, limit, count int, additionalQuery ...string) {

	format := func(additional []string) string {
		if len(additional) > 0 {
			return "&" + strings.Join(additional, "&")
		}
		return ""
	}

	// prev link
	if offset > 0 && count > 0 {
		var prevStart int
		// we do have a prev link
		if offset <= count {
			prevStart = offset - limit
		} else {
			// the first range that intersects the end of the useful range
			prevStart = offset - (((offset-count)/limit)+1)*limit
		}
		realLimit := limit
		if prevStart < 0 {
			// need to cut the range to start at 0
			realLimit = limit + prevStart
			prevStart = 0
		}
		prev := fmt.Sprintf("%s?page[offset]=%d&page[limit]=%d%s", path, prevStart, realLimit, format(additionalQuery))
		links.Prev = &prev
	}

	// next link
	nextStart := offset + resultLen
	if nextStart < count {
		// we have a next link
		next := fmt.Sprintf("%s?page[offset]=%d&page[limit]=%d%s", path, nextStart, limit, format(additionalQuery))
		links.Next = &next
	}

	// first link
	var firstEnd int
	if offset > 0 {
		firstEnd = offset % limit // this is where the second page starts
	} else {
		// offset == 0, first == current
		firstEnd = limit
	}
	first := fmt.Sprintf("%s?page[offset]=%d&page[limit]=%d%s", path, 0, firstEnd, format(additionalQuery))
	links.First = &first

	// last link
	var lastStart int
	if offset < count {
		// advance some pages until touching the end of the range
		lastStart = offset + (((count - offset - 1) / limit) * limit)
	} else {
		// retreat at least one page until covering the range
		lastStart = offset - ((((offset - count) / limit) + 1) * limit)
	}
	realLimit := limit
	if lastStart < 0 {
		// need to cut the range to start at 0
		realLimit = limit + lastStart
		lastStart = 0
	}
	last := fmt.Sprintf("%s?page[offset]=%d&page[limit]=%d%s", path, lastStart, realLimit, format(additionalQuery))
	links.Last = &last
}

func buildAbsoluteURL(req *goa.RequestData) string {
	return rest.AbsoluteURL(req, req.URL.Path)
}

func parseInts(s *string) ([]int, error) {
	if s == nil || len(*s) == 0 {
		return []int{}, nil
	}
	split := strings.Split(*s, ",")
	result := make([]int, len(split))
	for index, value := range split {
		converted, err := strconv.Atoi(value)
		if err != nil {
			return nil, errs.WithStack(err)
		}
		result[index] = converted
	}
	return result, nil
}

func parseLimit(pageParameter *string) (s *int, l int, e error) {
	params, err := parseInts(pageParameter)
	if err != nil {
		return nil, 0, errs.WithStack(err)
	}

	if len(params) > 1 {
		return &params[0], params[1], nil
	}
	if len(params) > 0 {
		return nil, params[0], nil
	}
	return nil, 100, nil
}
