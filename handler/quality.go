package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/guohuiyuan/go-music-api/service"
	"github.com/guohuiyuan/music-lib/model"
)

type qualitySourceResult struct {
	Source    string                     `json:"source"`
	Name      string                     `json:"name"`
	Qualities []model.AudioQualityOption `json:"qualities"`
	Available []model.AudioQualityOption `json:"available,omitempty"`
	Error     string                     `json:"error,omitempty"`
}

// GetAudioQualities returns static quality tiers and optionally probes the
// tiers currently available to the configured account.
// @Summary List audio quality tiers
// @Description Returns source quality tiers. Set probe=true to test the current account.
// @Tags Music
// @Produce json
// @Param source query string false "Music source"
// @Param sources query []string false "Music sources" collectionFormat(multi)
// @Param probe query bool false "Probe current account entitlement"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Router /api/v1/music/qualities [get]
func GetAudioQualities(c *gin.Context) {
	source := strings.ToLower(strings.TrimSpace(c.Query("source")))
	sources := c.QueryArray("sources")
	if source != "" {
		sources = []string{source}
	} else {
		sources = normalizedSources(sources, service.GetAudioQualitySourceNames())
	}
	if len(sources) == 0 {
		c.JSON(http.StatusBadRequest, Response{
			Code: http.StatusBadRequest,
			Msg:  "at least one source is required",
		})
		return
	}

	probe := strings.EqualFold(strings.TrimSpace(c.Query("probe")), "true") ||
		strings.TrimSpace(c.Query("probe")) == "1"
	if probe && len(sources) != 1 {
		c.JSON(http.StatusBadRequest, Response{
			Code: http.StatusBadRequest,
			Msg:  "probe requires exactly one source",
		})
		return
	}

	results := make([]qualitySourceResult, 0, len(sources))
	for _, currentSource := range sources {
		result := qualitySourceResult{
			Source: currentSource,
			Name:   service.GetSourceDescription(currentSource),
		}
		qualities, err := service.GetAudioQualityChoices(currentSource)
		if err != nil {
			result.Error = err.Error()
			results = append(results, result)
			continue
		}
		result.Qualities = qualities
		if probe {
			available, probeErr := service.GetAudioQualityOptions(currentSource)
			if probeErr != nil {
				result.Error = probeErr.Error()
			} else {
				result.Available = available
			}
		}
		results = append(results, result)
	}

	data := gin.H{"sources": results}
	if len(results) == 1 {
		data["source"] = results[0].Source
		data["qualities"] = results[0].Qualities
		if probe {
			data["available"] = results[0].Available
			if results[0].Error != "" {
				data["error"] = results[0].Error
			}
		}
	}
	c.JSON(http.StatusOK, Response{Code: http.StatusOK, Msg: "success", Data: data})
}
