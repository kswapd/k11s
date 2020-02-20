package ui

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/derailed/k9s/internal/config"
	"github.com/derailed/k9s/internal/render"
	"github.com/derailed/tview"
	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
)

// Synchronizer manages ui event queue.
type synchronizer interface {
	QueueUpdateDraw(func()) *tview.Application
	QueueUpdate(func()) *tview.Application
}

// Configurator represents an application configurationa.
type Configurator struct {
	skinFile  string
	Config    *config.Config
	Styles    *config.Styles
	BenchFile string
}

// HasSkin returns true if a skin file was located.
func (c *Configurator) HasSkin() bool {
	return c.skinFile != ""
}

// StylesUpdater watches for skin file changes.
func (c *Configurator) StylesUpdater(ctx context.Context, s synchronizer) error {
	if !c.HasSkin() {
		return nil
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case evt := <-w.Events:
				_ = evt
				s.QueueUpdateDraw(func() {
					c.RefreshStyles(c.Config.K9s.CurrentCluster)
				})
			case err := <-w.Errors:
				log.Info().Err(err).Msg("Skin watcher failed")
				return
			case <-ctx.Done():
				log.Debug().Msgf("SkinWatcher Done `%s!!", c.skinFile)
				if err := w.Close(); err != nil {
					log.Error().Err(err).Msg("Closing watcher")
				}
				return
			}
		}
	}()

	log.Debug().Msgf("SkinWatcher watching `%s", c.skinFile)
	return w.Add(c.skinFile)
}

// BenchConfig location of the benchmarks configuration file.
func BenchConfig(context string) string {
	return filepath.Join(config.K9sHome, config.K9sBench+"-"+context+".yml")
}

// RefreshStyles load for skin configuration changes.
func (c *Configurator) RefreshStyles(context string) {
	c.BenchFile = BenchConfig(context)

	clusterSkins := filepath.Join(config.K9sHome, fmt.Sprintf("%s_skin.yml", context))
	if c.Styles == nil {
		c.Styles = config.NewStyles()
	} else {
		c.Styles.Reset()
	}
	if err := c.Styles.Load(clusterSkins); err != nil {
		log.Info().Msgf("No context specific skin file found -- %s", clusterSkins)
	} else {
		c.updateStyles(clusterSkins)
		return
	}

	if err := c.Styles.Load(config.K9sStylesFile); err != nil {
		log.Info().Msgf("No skin file found -- %s. Loading stock skins.", config.K9sStylesFile)
		c.updateStyles("")
		return
	}
	c.updateStyles(config.K9sStylesFile)
}

func (c *Configurator) updateStyles(f string) {
	c.skinFile = f
	if !c.HasSkin() {
		c.Styles.DefaultSkin()
	}
	c.Styles.Update()

	render.StdColor = c.Styles.Frame().Status.NewColor.Color()
	render.AddColor = c.Styles.Frame().Status.AddColor.Color()
	render.ModColor = c.Styles.Frame().Status.ModifyColor.Color()
	render.ErrColor = c.Styles.Frame().Status.ErrorColor.Color()
	render.HighlightColor = c.Styles.Frame().Status.HighlightColor.Color()
	render.CompletedColor = c.Styles.Frame().Status.CompletedColor.Color()
}
