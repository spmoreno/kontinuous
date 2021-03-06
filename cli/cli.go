package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"net/http"

	apiReq "github.com/AcalephStorage/kontinuous/cli/request"
	"github.com/codegangsta/cli"
	"github.com/gosuri/uitable"
)

func main() {
	app := cli.NewApp()

	app.Name = "kontinuous-cli"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "conf, c",
			Value: "./config",
			Usage: "Specify an alternate configuratioxn file (default: ./config)",
		},
	}
	app.Commands = []cli.Command{
		{
			Name: "get",
			Subcommands: []cli.Command{
				{
					Name:      "pipelines",
					Usage:     "get all pipelines",
					ArgsUsage: "[pipeline-name]",
					Before: func(c *cli.Context) error {
						p := strings.TrimSpace(c.Args().First())
						if len(p) > 0 {
							return requireNameArg(c)
						}
						return nil
					},
					Action: getPipelines,
				},
				{
					Name:   "repos",
					Usage:  "get all repositories",
					Action: getRepos,
				},
				{
					Name:      "builds",
					Usage:     "get all builds of pipeline",
					ArgsUsage: "<pipeline-name>",
					Before:    requireNameArg,
					Action:    getBuilds,
				},
				{
					Name:      "stages",
					Usage:     "get the stages of a pipeline build",
					ArgsUsage: "<pipeline-name>",
					Before:    requireNameArg,
					Flags: []cli.Flag{
						cli.IntFlag{
							Name:  "build, b",
							Usage: "build number, if not provided will get stages of latest build",
						},
					},
					Action: getStages,
				},
			},
		},
		{
			Name: "create",
			Subcommands: []cli.Command{
				{
					Name:      "pipeline",
					Usage:     "create pipeline for repo",
					ArgsUsage: "<pipeline-name>",
					Before:    requireNameArg,
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "events",
							Value: "push",
						},
					},
					Action: createPipeline,
				},
				{
					Name:      "build",
					Usage:     "trigger pipeline build",
					ArgsUsage: "<pipeline-name>",
					Before:    requireNameArg,
					Action:    createBuild,
				},
			},
		},
		{
			Name: "delete",
			Subcommands: []cli.Command{
				{
					Name:      "pipeline",
					Usage:     "delete pipeline",
					ArgsUsage: "<pipeline-name>",
					Before:    requireNameArg,
					Action:    deletePipeline,
				},
				{
					Name:      "build",
					Usage:     "delete build",
					ArgsUsage: "<pipeline-name>",
					Flags: []cli.Flag{
						cli.IntFlag{
							Name:  "build, b",
							Usage: "build number, if not provided will not proceed deletion",
						},
					},
					Before: requireNameArg,
					Action: deleteBuild,
				},
			},
		},
		{
			Name:  "deploy",
			Usage: "deploy kontinuous app in the cluster",
			Subcommands: []cli.Command{
				{
					Name:   "remove",
					Usage:  "remove Kontinuous resources in the cluster",
					Action: removeDeployedApp,
				},
			},
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "namespace",
					Usage: "Required, Kubernetes namespace to deploy Kontinuous",
					Value: "kontinuous",
				},
				cli.StringFlag{
					Name:  "auth-secret",
					Usage: "Required, base64 encoded secret to sign JWT",
				},
				cli.StringFlag{
					Name:  "github-client-id",
					Usage: "Required, Github Client ID for Github authentication",
				},
				cli.StringFlag{
					Name:  "github-client-secret",
					Usage: "Required, Github Client Secret for Github authentication",
				},
			},

			Action: deployApp,
		},
		{
			Name:  "init",
			Usage: "create and save initial .pipeline.yml in your repository",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "namespace",
					Usage: "Required, Kubernetes namespace to deploy Kontinuous",
					Value: "kontinuous",
				},
				cli.StringFlag{
					Name:  "repository, repo, r",
					Usage: "Required, Github repository name ",
				},
				cli.StringFlag{
					Name:  "owner, o",
					Usage: "Required, Github owner name",
				},
			},

			Action: initialize,
		},
		{
			Name: "describe",
			Subcommands: []cli.Command{
				{
					Name:      "pipeline",
					Usage:     "display pipeline latest build information",
					ArgsUsage: "[pipeline-name]",
					Before: func(c *cli.Context) error {
						p := strings.TrimSpace(c.Args().First())
						if len(p) > 0 {
							return requireNameArg(c)
						}
						return nil
					},
					Action: getLatestBuild,
				},
			},
		},
		{
			Name:      "resume",
			Usage:     "resume pipeline builds",
			ArgsUsage: "[pipeline-name]",
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  "build, b",
					Usage: "Required, Pipeline build number you want to resume",
				},
			},
			Before: func(c *cli.Context) error {
				p := strings.TrimSpace(c.Args().First())
				if len(p) > 0 {
					return requireNameArg(c)
				}
				return nil
			},
			Action: resumeBuild,
		},
	}
	app.Run(os.Args)
}

func requireNameArg(c *cli.Context) error {
	if _, _, err := parseNameArg(c.Args().First()); err != nil {
		return err
	}
	return nil
}

func parseNameArg(name string) (owner, repo string, err error) {
	invalid := errors.New("Invalid pipeline name")
	required := errors.New("Provide pipeline name")

	p := strings.TrimSpace(name)
	if len(p) == 0 {
		return "", "", required
	}
	fullName := strings.Split(p, "/")
	if len(fullName) != 2 {
		return "", "", invalid
	}

	owner = fullName[0]
	repo = fullName[1]
	if len(owner) == 0 || len(repo) == 0 {
		return "", "", invalid
	}
	return owner, repo, nil
}

// ACTIONS

func getPipelines(c *cli.Context) {
	config, err := apiReq.GetConfigFromFile(c.GlobalString("conf"))
	if err != nil {
		os.Exit(1)
	}
	pipelineName := c.Args().First()
	pipelines, err := config.GetPipelines(http.DefaultClient, pipelineName)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	table := uitable.New()
	table.AddRow("NAME")
	for _, p := range pipelines {
		name := fmt.Sprintf("%s/%s", p.Owner, p.Repo)
		table.AddRow(name)
	}
	fmt.Println(table)
}

func getRepos(c *cli.Context) {
	config, err := apiReq.GetConfigFromFile(c.GlobalString("conf"))
	if err != nil {
		os.Exit(1)
	}
	repos, err := config.GetRepos(http.DefaultClient)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	table := uitable.New()
	table.AddRow("OWNER", "NAME")
	for _, r := range repos {
		table.AddRow(r.Owner, r.Name)
	}
	fmt.Println(table)
}

func getBuilds(c *cli.Context) {
	config, err := apiReq.GetConfigFromFile(c.GlobalString("conf"))
	if err != nil {
		os.Exit(1)
	}

	owner, repo, _ := parseNameArg(c.Args().First())
	builds, err := config.GetBuilds(http.DefaultClient, owner, repo)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	table := uitable.New()
	table.AddRow("BUILD", "STATUS", "CREATED", "FINISHED", "EVENT", "AUTHOR", "COMMIT")
	for _, b := range builds {
		created := "-"
		if b.Created != 0 {
			created = time.Unix(0, b.Created).Format(time.RFC3339)
		}
		finished := "-"
		if b.Finished != 0 {
			finished = time.Unix(0, b.Finished).Format(time.RFC3339)
		}
		table.AddRow(b.Number, b.Status, created, finished, b.Event, b.Author, b.Commit)
	}
	fmt.Println(table)
}

func getLatestBuild(c *cli.Context) {
	config, err := apiReq.GetConfigFromFile(c.GlobalString("conf"))
	if err != nil {
		os.Exit(1)
	}

	owner, repo, _ := parseNameArg(c.Args().First())
	pipelineName := fmt.Sprintf("%s/%s", owner, repo)

	pipeline, err := config.GetPipeline(http.DefaultClient, pipelineName)
	if err != nil {
		fmt.Printf("Pipeline %s doesn't exist \n", pipelineName)
		os.Exit(1)
	}

	stages, _ := config.GetStages(http.DefaultClient, owner, repo, pipeline.LatestBuild.Number)

	table := uitable.New()
	table.Wrap = true

	if pipeline.LatestBuild == nil {
		table.AddRow("No available builds for pipeline", pipelineName)
		fmt.Println(table)
		os.Exit(1)
	}

	started := time.Unix(0, pipeline.LatestBuild.Created).Format(time.RFC3339)
	finished := time.Unix(0, pipeline.LatestBuild.Finished).Format(time.RFC3339)

	table.AddRow("")
	table.AddRow("Name:", pipelineName)
	table.AddRow("Build No:", pipeline.LatestBuild.Number)
	table.AddRow("Status:", pipeline.LatestBuild.Status)
	table.AddRow("Author:", pipeline.LatestBuild.Author)
	table.AddRow("Started:", started)
	table.AddRow("Finished:", finished)
	table.AddRow("No of Stages:", len(stages))
	table.AddRow("")
	fmt.Println(table)

}

func getStages(c *cli.Context) {
	config, err := apiReq.GetConfigFromFile(c.GlobalString("conf"))
	if err != nil {
		os.Exit(1)
	}
	owner, repo, _ := parseNameArg(c.Args().First())
	stages, err := config.GetStages(http.DefaultClient, owner, repo, c.Int("build"))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	table := uitable.New()
	table.AddRow("INDEX", "TYPE", "NAME", "STATUS", "STARTED", "FINISHED")
	for _, s := range stages {
		started := "-"
		if s.Started != 0 {
			started = time.Unix(0, s.Started).Format(time.RFC3339)
		}
		finished := "-"
		if s.Finished != 0 {
			finished = time.Unix(0, s.Finished).Format(time.RFC3339)
		}
		table.AddRow(s.Index, s.Type, s.Name, s.Status, started, finished)
	}
	fmt.Println(table)
}

func createPipeline(c *cli.Context) {
	config, err := apiReq.GetConfigFromFile(c.GlobalString("conf"))
	if err != nil {
		os.Exit(1)
	}
	owner, repo, _ := parseNameArg(c.Args().First())
	events := strings.Split(c.String("events"), ",")
	for i, e := range events {
		events[i] = strings.TrimSpace(e)
	}

	pipeline := &apiReq.PipelineData{
		Owner:  owner,
		Repo:   repo,
		Events: events,
	}

	err = config.CreatePipeline(http.DefaultClient, pipeline)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("pipeline `%s/%s` created", pipeline.Owner, pipeline.Repo)
	}
}

func createBuild(c *cli.Context) {
	config, err := apiReq.GetConfigFromFile(c.GlobalString("conf"))
	if err != nil {
		os.Exit(1)
	}
	owner, repo, _ := parseNameArg(c.Args().First())
	err = config.CreateBuild(http.DefaultClient, owner, repo)

	if err != nil {
		fmt.Println(err)
	}
}

func deletePipeline(c *cli.Context) {
	config, err := apiReq.GetConfigFromFile(c.GlobalString("conf"))
	if err != nil {
		os.Exit(1)
	}
	pipelineName := c.Args().First()
	err = config.DeletePipeline(http.DefaultClient, pipelineName)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("pipeline %s successfully deleted.\n", pipelineName)
}

func deleteBuild(c *cli.Context) {
	config, err := apiReq.GetConfigFromFile(c.GlobalString("conf"))
	if err != nil {
		os.Exit(1)
	}

	pipelineName := c.Args().First()
	buildNum := c.String("build")
	err = config.DeleteBuild(http.DefaultClient, pipelineName, buildNum)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("pipeline %s build #%s successfully deleted.\n", pipelineName, buildNum)
}

func deployApp(c *cli.Context) {
	namespace := c.String("namespace")
	authCode := c.String("auth-secret")
	clientId := c.String("github-client-id")
	clientSecret := c.String("github-client-secret")

	missingFields := false
	if namespace == "" || authCode == "" || clientId == "" || clientSecret == "" {
		missingFields = true
	}

	if !missingFields {
		err := DeployKontinuous(namespace, authCode, clientId, clientSecret)
		if err != nil {
			fmt.Println("Missing fields. Unable to deploy Kontinuous.")
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("Success! Kontinuous is now deployed in the cluster.")
	}
}

func initialize(c *cli.Context) {
	config, err := apiReq.GetConfigFromFile(c.GlobalString("conf"))
	if err != nil {
		os.Exit(1)
	}
	namespace := c.String("namespace")
	owner := c.String("owner")
	repository := c.String("repository")
	token := config.Token

	missingFields := false
	if namespace == "" || owner == "" || repository == "" {
		missingFields = true
	}

	if !missingFields {
		err := Init(namespace, owner, repository, token)
		if err != nil {
			fmt.Println("Missing fields. Unable to initialize Kontinuous in the repository")
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("Success! Initialization complete.")
	}
}

func removeDeployedApp(c *cli.Context) {
	err := RemoveResources()

	if err != nil {
		fmt.Println("Unable to remove kontinuous.")
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("Success! Kontinuous resources has been removed from the cluster. ")
}

func resumeBuild(c *cli.Context) {
	config, err := apiReq.GetConfigFromFile(c.GlobalString("conf"))
	if err != nil {
		os.Exit(1)
	}
	owner, repo, _ := parseNameArg(c.Args().First())
	buildNo := c.Int("build")

	if owner == "" || repo == "" || buildNo == 0 {
		fmt.Println("Missing fields.")
		os.Exit(1)
	}

	err = config.ResumeBuild(http.DefaultClient, owner, repo, buildNo)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
