package handler

import (
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/guohuiyuan/go-music-api/service"
	"github.com/guohuiyuan/music-lib/model"
)

type rankingSourceResult struct {
	Source   string          `json:"source"`
	Name     string          `json:"name"`
	TopLists []model.TopList `json:"toplists,omitempty"`
	Error    string          `json:"error,omitempty"`
}

// GetTopLists returns the chart catalogs exposed by the requested providers.
// @Summary List ranking charts
// @Description Returns ranking chart metadata for one or more music sources.
// @Tags Ranking
// @Produce json
// @Param sources query []string false "Music sources" collectionFormat(multi)
// @Success 200 {object} Response
// @Router /api/v1/ranking/toplists [get]
func GetTopLists(c *gin.Context) {
	sources := normalizedSources(c.QueryArray("sources"), service.GetTopListSourceNames())
	results := make([]rankingSourceResult, len(sources))
	var wg sync.WaitGroup

	for index, source := range sources {
		results[index] = rankingSourceResult{
			Source: source,
			Name:   service.GetSourceDescription(source),
		}
		fn := service.GetTopListsFunc(source)
		if fn == nil {
			results[index].Error = "unsupported source"
			continue
		}

		wg.Add(1)
		go func(index int, source string, fn service.TopListsFunc) {
			defer wg.Done()
			topLists, err := fn()
			if err != nil {
				results[index].Error = err.Error()
				return
			}
			for i := range topLists {
				if strings.TrimSpace(topLists[i].Source) == "" {
					topLists[i].Source = source
				}
			}
			results[index].TopLists = topLists
		}(index, source, fn)
	}
	wg.Wait()

	allTopLists := make([]model.TopList, 0)
	for _, result := range results {
		allTopLists = append(allTopLists, result.TopLists...)
	}
	c.JSON(http.StatusOK, Response{
		Code: http.StatusOK,
		Msg:  "success",
		Data: gin.H{
			"toplists": allTopLists,
			"sources":  results,
		},
	})
}

// GetTopListSongs returns all songs in a ranking chart.
// @Summary Get ranking chart songs
// @Description Returns the ranked songs for one source and chart ID.
// @Tags Ranking
// @Produce json
// @Param source query string true "Music source"
// @Param toplist_id query string true "Ranking chart ID"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Router /api/v1/ranking/toplist [get]
func GetTopListSongs(c *gin.Context) {
	source := strings.ToLower(strings.TrimSpace(c.Query("source")))
	topListID := strings.TrimSpace(c.Query("toplist_id"))
	if topListID == "" {
		topListID = strings.TrimSpace(c.Query("top_list_id"))
	}
	if topListID == "" {
		topListID = strings.TrimSpace(c.Query("id"))
	}
	if source == "" || topListID == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code: http.StatusBadRequest,
			Msg:  "source and toplist_id are required",
		})
		return
	}

	fn := service.GetTopListSongsFunc(source)
	if fn == nil {
		c.JSON(http.StatusBadRequest, Response{
			Code: http.StatusBadRequest,
			Msg:  "source does not support ranking charts",
		})
		return
	}
	songs, err := fn(topListID)
	if err != nil {
		c.JSON(http.StatusBadGateway, Response{
			Code: http.StatusBadGateway,
			Msg:  err.Error(),
		})
		return
	}
	for index := range songs {
		if strings.TrimSpace(songs[index].Source) == "" {
			songs[index].Source = source
		}
	}

	c.JSON(http.StatusOK, Response{
		Code: http.StatusOK,
		Msg:  "success",
		Data: gin.H{
			"source":     source,
			"toplist_id": topListID,
			"songs":      songs,
		},
	})
}

// GetHotSongs returns the provider's current hot-song chart.
// @Summary Get hot songs
// @Description Returns the current hot songs from one source.
// @Tags Ranking
// @Produce json
// @Param source query string true "Music source"
// @Param limit query int false "Number of songs" default(50)
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Router /api/v1/ranking/hot [get]
func GetHotSongs(c *gin.Context) {
	source := strings.ToLower(strings.TrimSpace(c.Query("source")))
	if source == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code: http.StatusBadRequest,
			Msg:  "source is required",
		})
		return
	}
	limit := parsePositiveIntQuery(c, "limit", 50)
	if limit > 500 {
		limit = 500
	}

	fn := service.GetHotSongsFunc(source)
	if fn == nil {
		c.JSON(http.StatusBadRequest, Response{
			Code: http.StatusBadRequest,
			Msg:  "source does not support hot songs",
		})
		return
	}
	songs, err := fn(limit)
	if err != nil {
		c.JSON(http.StatusBadGateway, Response{
			Code: http.StatusBadGateway,
			Msg:  err.Error(),
		})
		return
	}
	for index := range songs {
		if strings.TrimSpace(songs[index].Source) == "" {
			songs[index].Source = source
		}
	}

	c.JSON(http.StatusOK, Response{
		Code: http.StatusOK,
		Msg:  "success",
		Data: gin.H{
			"source": source,
			"limit":  limit,
			"songs":  songs,
		},
	})
}

func normalizedSources(requested, fallback []string) []string {
	if len(requested) == 0 {
		return append([]string(nil), fallback...)
	}
	result := make([]string, 0, len(requested))
	seen := make(map[string]struct{}, len(requested))
	for _, source := range requested {
		source = strings.ToLower(strings.TrimSpace(source))
		if source == "" {
			continue
		}
		if _, exists := seen[source]; exists {
			continue
		}
		seen[source] = struct{}{}
		result = append(result, source)
	}
	return result
}
