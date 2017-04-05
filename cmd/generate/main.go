package main

import (
	"flag"
	"log"
	"path"

	"os"

	"github.com/ubuntu/tutorial-deployment/apis"
	"github.com/ubuntu/tutorial-deployment/codelab"
	"github.com/ubuntu/tutorial-deployment/consts"
	"github.com/ubuntu/tutorial-deployment/internaltools"
	"github.com/ubuntu/tutorial-deployment/paths"
)

func main() {
	flag.Parse()
	args := internaltools.UniqueStrings(flag.Args())

	p := paths.New()
	if err := p.DetectPaths(); err != nil {
		log.Fatalf("Couldn't detect required paths: %s", err)
	}
	if err := p.ImportTutorialPaths(args); err != nil {
		log.Fatalf("Couldn't load tutorial paths: %s", err)
	}

	template := path.Join(p.MetaData, consts.TemplateFileName)

	type result struct {
		c   codelab.Codelab
		err error
	}
	ch := make(chan result)
	// export codelabs
	codelabRefs, err := codelab.Discover()
	if err != nil {
		log.Fatalf("Couldn't detect codelabs: %s", err)
	}
	if err := os.RemoveAll(p.Export); err != nil {
		log.Fatalf("Couldn't remove codelab export path %s: %v", p.Export, err)
	}
	for _, src := range codelabRefs {
		go func(ref string) {
			c, err := codelab.New(ref, p.Export, template, false)
			if err != nil {
				c = &codelab.Codelab{RefURI: ref}
			}
			ch <- result{*c, err}
		}(src)
	}

	hasError := false
	var codelabs []codelab.Codelab
	for _ = range codelabRefs {
		res := <-ch
		if res.err != nil {
			log.Printf("ERROR in %s: %v", res.c.RefURI, res.err)
			hasError = true
			continue
		}
		codelabs = append(codelabs, res.c)
	}
	if hasError {
		os.Exit(1)
	}

	if err := os.RemoveAll(p.API); err != nil {
		log.Fatalf("Couldn't remove API export path %s: %v", p.API, err)
	}
	dat, err := apis.GenerateContent(codelabs)
	if err != nil {
		log.Fatalf("Couldn't generate API: %s", err)
	}
	if err != apis.Save(dat) {
		log.Fatalf("Coudln't save API: %s", err)
	}
}
