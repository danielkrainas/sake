package service

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
		Name:        "More Complicated Test Flow",
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
			},

			"end": &Stage{
				Terminate: true,
			},
		},
	},
}
