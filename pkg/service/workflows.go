package service

import "time"

var Workflows = []*Workflow{
	{
		Name:        "Test Flow",
		TriggeredBy: "init-start",
		StartAt:     "start",
		Stages: map[string]*Stage{
			"start": &Stage{
				Next:     "end",
				Rollback: "cancel-start",
			},

			"end": &Stage{
				Terminate: true,
			},
		},
	},
	{
		Name:        "complicated-test",
		TriggeredBy: "init-start",
		StartAt:     "start",
		Stages: map[string]*Stage{
			"start": &Stage{
				Next:     "middle",
				Rollback: "cancel-start",
			},

			"middle": &Stage{
				Next:     "end",
				Rollback: "cancel-middle",
				Timeout:  2 * time.Second,
			},

			"end": &Stage{
				Terminate: true,
			},
		},
	},
}
