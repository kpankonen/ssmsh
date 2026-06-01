package parameterstore_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/bwhaley/ssmsh/parameterstore"
)

var EddardStark = types.Parameter{
	Name:  aws.String("/House/Stark/EddardStark"),
	Type:  types.ParameterTypeString,
	Value: aws.String("Lord"),
}

var CatelynStark = types.Parameter{
	Name:  aws.String("/House/Stark/CatelynStark"),
	Type:  types.ParameterTypeString,
	Value: aws.String("Lady"),
}

var RobStark = types.Parameter{
	Name:  aws.String("/House/Stark/RobStark"),
	Type:  types.ParameterTypeString,
	Value: aws.String("Noble"),
}

var JonSnow = types.Parameter{
	Name:  aws.String("/House/Stark/JonSnow"),
	Type:  types.ParameterTypeString,
	Value: aws.String("Bastard"),
}

var DaenerysTargaryen = types.Parameter{
	Name:  aws.String("/House/Targaryen/DaenerysTargaryen"),
	Type:  types.ParameterTypeString,
	Value: aws.String("Noble"),
}

var HouseStark = []types.Parameter{
	EddardStark,
	CatelynStark,
	RobStark,
}

var HouseTargaryen = []types.Parameter{
	DaenerysTargaryen,
}

const NextToken = "A1B2C3D4"

type mockedSSM struct {
	GetParametersByPathResp ssm.GetParametersByPathOutput
	GetParametersByPathNext ssm.GetParametersByPathOutput
	GetParameterHistoryResp ssm.GetParameterHistoryOutput
	GetParametersResp       ssm.GetParametersOutput
	GetParameterResp        []ssm.GetParameterOutput
	DeleteParametersResp    ssm.DeleteParametersOutput
	PutParameterResp        ssm.PutParameterOutput
}

func (m mockedSSM) GetParametersByPath(_ context.Context, in *ssm.GetParametersByPathInput, _ ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error) {
	if in.NextToken != nil {
		return &m.GetParametersByPathNext, nil
	}
	return &m.GetParametersByPathResp, nil
}

func (m mockedSSM) DeleteParameters(_ context.Context, in *ssm.DeleteParametersInput, _ ...func(*ssm.Options)) (*ssm.DeleteParametersOutput, error) {
	return &m.DeleteParametersResp, nil
}

func (m mockedSSM) GetParameter(_ context.Context, in *ssm.GetParameterInput, _ ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	parameterName := aws.ToString(in.Name)
	for _, param := range m.GetParameterResp {
		if aws.ToString(param.Parameter.Name) == parameterName {
			return &param, nil
		}
	}
	return nil, errors.New("Parameter not found")
}

func (m mockedSSM) GetParameterHistory(_ context.Context, in *ssm.GetParameterHistoryInput, _ ...func(*ssm.Options)) (*ssm.GetParameterHistoryOutput, error) {
	return &m.GetParameterHistoryResp, nil
}

func (m mockedSSM) GetParameters(_ context.Context, in *ssm.GetParametersInput, _ ...func(*ssm.Options)) (*ssm.GetParametersOutput, error) {
	for _, n := range in.Names {
		input := &ssm.GetParameterInput{
			Name:           aws.String(n),
			WithDecryption: aws.Bool(true),
		}
		parameter, err := m.GetParameter(context.TODO(), input)
		if err != nil {
			m.GetParametersResp.InvalidParameters = append(m.GetParametersResp.InvalidParameters, n)
		} else {
			m.GetParametersResp.Parameters = append(m.GetParametersResp.Parameters, *parameter.Parameter)
		}
	}
	return &m.GetParametersResp, nil
}

func (m mockedSSM) PutParameter(_ context.Context, in *ssm.PutParameterInput, _ ...func(*ssm.Options)) (*ssm.PutParameterOutput, error) {
	return &m.PutParameterResp, nil
}

func TestPut(t *testing.T) {
	var expectedVersion int64 = 1
	var p parameterstore.ParameterStore
	err := p.NewParameterStore(false)
	if err != nil {
		t.Fatal(err)
	}
	p.Cwd = parameterstore.Delimiter
	p.Clients[p.Region] = mockedSSM{
		PutParameterResp: ssm.PutParameterOutput{
			Version: expectedVersion,
		},
	}
	putParameterInput := ssm.PutParameterInput{
		Name:        aws.String("/House/Stark/EddardStark"),
		Value:       aws.String("Lord"),
		Description: aws.String("Lord of Winterfell in Season 1"),
		Type:        types.ParameterTypeString,
	}
	resp, err := p.Put(&putParameterInput, p.Region)
	if err != nil {
		t.Fatal("Error putting parameter", err)
	} else {
		if resp.Version != expectedVersion {
			msg := fmt.Errorf("expected %d, got %d", expectedVersion, resp.Version)
			t.Fatal(msg)
		}
	}
}

func TestMoveParameter(t *testing.T) {
	srcParam := parameterstore.ParameterPath{
		Name:   "/House/Stark/SansaStark",
		Region: "region",
	}
	dstParam := parameterstore.ParameterPath{
		Name:   "/House/Lannister/SansaStark",
		Region: "region",
	}
	var p parameterstore.ParameterStore
	p.Region = "region"
	err := p.NewParameterStore(false)
	if err != nil {
		t.Fatal(err)
	}
	p.Cwd = parameterstore.Delimiter
	p.Clients[p.Region] = mockedSSM{
		GetParameterResp: []ssm.GetParameterOutput{
			{
				Parameter: &types.Parameter{
					Name:  aws.String(srcParam.Name),
					Type:  types.ParameterTypeString,
					Value: aws.String("Noble"),
				},
			},
			{
				Parameter: &types.Parameter{
					Name:  aws.String(dstParam.Name),
					Type:  types.ParameterTypeString,
					Value: aws.String("Noble"),
				},
			},
		},
		GetParameterHistoryResp: ssm.GetParameterHistoryOutput{
			Parameters: []types.ParameterHistory{
				{
					Name:        aws.String(srcParam.Name),
					Value:       aws.String("Noble"),
					Type:        types.ParameterTypeString,
					Description: aws.String("Eldest daughter of House Stark, bethrothed to Tyrion Lannister"),
					Version:     2,
				},
				{
					Name:        aws.String(srcParam.Name),
					Value:       aws.String("Noble"),
					Type:        types.ParameterTypeString,
					Description: aws.String("Eldest daughter of House Stark"),
					Version:     1,
				},
			},
		},
	}
	err = p.Move(srcParam, dstParam)
	if err != nil {
		t.Fatal("Error moving parameter", err)
	}
	p.Clients[p.Region] = mockedSSM{
		GetParameterResp: []ssm.GetParameterOutput{
			{
				Parameter: &types.Parameter{
					Name:  aws.String(dstParam.Name),
					Type:  types.ParameterTypeString,
					Value: aws.String("Noble"),
				},
			},
		},
	}
	resp, err := p.Get([]string{srcParam.Name}, p.Region)
	if err != nil {
		msg := fmt.Errorf("Error getting %s: %s", srcParam.Name, err)
		t.Fatal(msg)
	}
	if len(resp) > 0 {
		if err != nil {
			msg := fmt.Errorf("Expected parameter %s to be removed but found %v", srcParam.Name, resp)
			t.Fatal(msg)
		}
	}
	_, err = p.Get([]string{dstParam.Name}, p.Region)
	if err != nil {
		msg := fmt.Errorf("Expected to find %s but didn't", dstParam.Name)
		t.Fatal(msg)
	}
}

func TestCopyPath(t *testing.T) {
	srcPath := parameterstore.ParameterPath{
		Name:   "/House/Stark",
		Region: "region",
	}
	dstPath := parameterstore.ParameterPath{
		Name:   "/House/Targaryen",
		Region: "region",
	}

	var p parameterstore.ParameterStore
	p.Region = "region"
	err := p.NewParameterStore(false)
	if err != nil {
		t.Fatal(err)
	}
	p.Cwd = parameterstore.Delimiter
	bothHouses := append(HouseStark, HouseTargaryen...)
	p.Clients[p.Region] = mockedSSM{
		GetParameterResp: []ssm.GetParameterOutput{
			{Parameter: &EddardStark},
			{Parameter: &CatelynStark},
			{Parameter: &RobStark},
			{Parameter: &JonSnow},
			{Parameter: &DaenerysTargaryen},
		},
		GetParametersByPathResp: ssm.GetParametersByPathOutput{
			Parameters: bothHouses,
			NextToken:  aws.String(NextToken),
		},
		GetParametersByPathNext: ssm.GetParametersByPathOutput{
			Parameters: []types.Parameter{JonSnow},
		},
		GetParameterHistoryResp: ssm.GetParameterHistoryOutput{
			Parameters: []types.ParameterHistory{
				{
					Name:    aws.String("/House/Stark/EddardStark"),
					Version: 2,
				},
			},
		},
	}
	err = p.Copy(srcPath, dstPath, true)
	if err != nil {
		t.Fatal("Error copying parameter path: ", err)
	}
	expectedName := parameterstore.ParameterPath{
		Name:   "/House/Targaryen/Stark/EddardStark",
		Region: "region",
	}
	resp, err := p.GetHistory(expectedName)
	if err != nil {
		t.Fatal("Error getting history: ", err)
	}
	if len(resp) != 1 {
		msg := fmt.Errorf("Expected history of length 1, got %v", resp)
		t.Fatal(msg)
	}
}

func TestCopyParameter(t *testing.T) {
	srcParam := parameterstore.ParameterPath{
		Name:   "/House/Stark/JonSnow",
		Region: "region",
	}
	dstParam := parameterstore.ParameterPath{
		Name:   "/House/Targaryen/JonSnow",
		Region: "region",
	}
	var p parameterstore.ParameterStore
	p.Region = "region"
	err := p.NewParameterStore(false)
	if err != nil {
		t.Fatal(err)
	}
	p.Cwd = parameterstore.Delimiter
	p.Clients[p.Region] = mockedSSM{
		GetParameterResp: []ssm.GetParameterOutput{
			{
				Parameter: &types.Parameter{
					Name:  aws.String("/House/Stark/JonSnow"),
					Type:  types.ParameterTypeString,
					Value: aws.String("King"),
				},
			},
			{
				Parameter: &types.Parameter{
					Name:  aws.String("/House/Targaryen/JonSnow"),
					Type:  types.ParameterTypeString,
					Value: aws.String("King"),
				},
			},
		},
		GetParameterHistoryResp: ssm.GetParameterHistoryOutput{
			Parameters: []types.ParameterHistory{
				{
					Name:        aws.String("/House/Stark/JonSnow"),
					Value:       aws.String("King"),
					Type:        types.ParameterTypeString,
					Description: aws.String("King of the north"),
					Version:     2,
				},
				{
					Name:        aws.String("/House/Stark/JonSnow"),
					Value:       aws.String("Bastard"),
					Type:        types.ParameterTypeString,
					Description: aws.String("Bastard of Winterfell"),
					Version:     1,
				},
			},
		},
	}
	err = p.Copy(srcParam, dstParam, false)
	if err != nil {
		t.Fatal("Error copying parameter", err)
	}
	resp, err := p.Get([]string{dstParam.Name}, p.Region)
	if err != nil {
		t.Fatal("Error getting parameter", err)
	}
	expectedName := parameterstore.ParameterPath{
		Name:   "/House/Targaryen/JonSnow",
		Region: "region",
	}
	if aws.ToString(resp[0].Name) != expectedName.Name {
		msg := fmt.Errorf("expected %s, got %s", expectedName.Name, aws.ToString(resp[0].Name))
		t.Fatal(msg)
	}
}

func TestCwd(t *testing.T) {
	cases := []struct {
		GetParametersByPathResp ssm.GetParametersByPathOutput
		Path                    string
		Expected                string
	}{
		{
			Path:     "/",
			Expected: "/",
		},
		{
			Path: "/House/Stark/..///Deceased",
			GetParametersByPathResp: ssm.GetParametersByPathOutput{
				Parameters: []types.Parameter{
					{
						Name:  aws.String("/House/Stark/EddardStark"),
						Type:  types.ParameterTypeString,
						Value: aws.String("Lord"),
					},
				},
			},
			Expected: "/House/Deceased",
		},
	}

	var p parameterstore.ParameterStore
	for _, c := range cases {
		err := p.NewParameterStore(false)
		if err != nil {
			t.Fatal("unexpected error", err)
		}
		p.Region = "region"
		p.Cwd = parameterstore.Delimiter
		p.Clients[p.Region] = mockedSSM{
			GetParametersByPathResp: c.GetParametersByPathResp,
		}
		err = p.SetCwd(parameterstore.ParameterPath{Name: c.Path, Region: "region"})
		if err != nil {
			t.Fatal("unexpected error", err)
		}
		if p.Cwd != c.Expected {
			msg := fmt.Errorf("expected %v, got %v", c.Expected, p.Cwd)
			t.Fatal(msg)
		}
	}

	err := p.NewParameterStore(false)
	if err != nil {
		t.Fatal("unexpected error", err)
	}
	p.Cwd = parameterstore.Delimiter
	testDir := parameterstore.ParameterPath{
		Name:   "/nodir",
		Region: "region",
	}
	err = p.SetCwd(testDir)
	if err == nil {
		msg := fmt.Errorf("Expected error for dir %s, got cwd %s ", testDir, p.Cwd)
		t.Fatal(msg)
	}
}

func TestDelete(t *testing.T) {
	testParams := []parameterstore.ParameterPath{
		{
			Name:   "/House/Stark/EddardStark",
			Region: "region",
		},
		{
			Name:   "/House/Stark/CatelynStark",
			Region: "region",
		},
		{
			Name:   "/House/Stark/TyrionLannister",
			Region: "region",
		},
	}
	deleteParametersOutput := ssm.DeleteParametersOutput{
		DeletedParameters: []string{
			"/House/Stark/EddardStark",
			"/House/Stark/CatelynStark",
		},
		InvalidParameters: []string{
			"/House/Stark/TyrionLannister",
		},
	}

	var p parameterstore.ParameterStore
	p.Region = "region"
	err := p.NewParameterStore(false)
	if err != nil {
		t.Fatal("unexpected error", err)
	}
	p.Clients[p.Region] = mockedSSM{
		DeleteParametersResp: deleteParametersOutput,
	}
	err = p.Remove(testParams, false)
	if err == nil {
		msg := fmt.Errorf("Expected error for param %s, got %v ", testParams[2], err)
		t.Fatal(msg)
	}
}

func TestGetHistory(t *testing.T) {
	testParam := parameterstore.ParameterPath{
		Name:   "/House/Stark/EddardStark",
		Region: "region",
	}
	getHistoryOutput := ssm.GetParameterHistoryOutput{
		Parameters: []types.ParameterHistory{
			{
				Name:    aws.String("/House/Stark/EddardStark"),
				Version: 2,
			},
			{
				Name:    aws.String("/House/Stark/EddardStark"),
				Version: 1,
			},
		},
	}
	var p parameterstore.ParameterStore
	p.Region = "region"
	err := p.NewParameterStore(false)
	if err != nil {
		t.Fatal("unexpected error", err)
	}
	p.Clients[p.Region] = mockedSSM{
		GetParameterHistoryResp: getHistoryOutput,
	}
	resp, err := p.GetHistory(testParam)
	if err != nil {
		msg := fmt.Errorf("Unexpected error %s", err)
		t.Fatal(msg)
	}
	if len(resp) != 2 {
		msg := fmt.Errorf("Expected history of length 2, got %v", resp)
		t.Fatal(msg)
	}
}

func TestList(t *testing.T) {
	cases := []struct {
		Query                   parameterstore.ParameterPath
		GetParametersByPathResp ssm.GetParametersByPathOutput
		GetParametersResp       ssm.GetParametersOutput
		GetParametersByPathNext ssm.GetParametersByPathOutput
		Expected                []string
		Recurse                 bool
	}{
		{
			Query: parameterstore.ParameterPath{
				Name:   "/House/Stark/EddardStark",
				Region: "region",
			},
			Recurse: false,
			GetParametersByPathResp: ssm.GetParametersByPathOutput{
				Parameters: []types.Parameter{},
			},
			Expected: []string{
				"/House/Stark/EddardStark",
			},
			GetParametersResp: ssm.GetParametersOutput{
				Parameters: []types.Parameter{
					{
						Name:  aws.String("/House/Stark/EddardStark"),
						Type:  types.ParameterTypeString,
						Value: aws.String("Lord"),
					},
				},
			},
		}, {
			Query: parameterstore.ParameterPath{
				Name:   "/",
				Region: "region",
			},
			Recurse: false,
			Expected: []string{
				"root",
			},
			GetParametersResp: ssm.GetParametersOutput{
				Parameters: []types.Parameter{
					{
						Name:  aws.String("root"),
						Type:  types.ParameterTypeString,
						Value: aws.String("A root parameter"),
					},
				},
			},
		},
		{
			Query: parameterstore.ParameterPath{
				Name:   "/House/Stark",
				Region: "region",
			},
			Recurse: false,
			GetParametersByPathResp: ssm.GetParametersByPathOutput{
				Parameters: HouseStark,
			},
			Expected: []string{
				"EddardStark",
				"CatelynStark",
				"RobStark",
			},
		},
		{
			Query: parameterstore.ParameterPath{
				Name:   "/House/",
				Region: "region",
			},
			Recurse: true,
			GetParametersByPathResp: ssm.GetParametersByPathOutput{
				Parameters: HouseStark,
				NextToken:  aws.String(NextToken),
			},
			GetParametersByPathNext: ssm.GetParametersByPathOutput{
				Parameters: []types.Parameter{JonSnow, DaenerysTargaryen},
			},
			Expected: []string{
				"/House/Stark/EddardStark",
				"/House/Stark/CatelynStark",
				"/House/Stark/RobStark",
				"/House/Stark/JonSnow",
				"/House/Targaryen/DaenerysTargaryen",
			},
		},
	}

	for _, c := range cases {
		var p parameterstore.ParameterStore
		p.Region = "region"
		err := p.NewParameterStore(false)
		if err != nil {
			t.Fatal("unexpected error", err)
		}
		p.Clients[p.Region] = mockedSSM{
			GetParametersByPathResp: c.GetParametersByPathResp,
			GetParametersByPathNext: c.GetParametersByPathNext,
			GetParametersResp:       c.GetParametersResp,
		}
		p.Cwd = parameterstore.Delimiter

		ch := make(chan parameterstore.ListResult)
		quit := make(chan bool)
		go func() {
			p.List(c.Query, c.Recurse, ch, quit)
		}()

		result := <-ch
		if result.Error != nil {
			quit <- true
			t.Fatal("unexpected error", result.Error)
		}
		if !equal(result.Result, c.Expected) {
			msg := fmt.Errorf("expected %v, got %v", c.Expected, result.Result)
			t.Fatal(msg)
		}
	}
}

func equal(first []string, second []string) bool {
	if len(first) != len(second) {
		return false
	}
	for i := 0; i < len(first); i++ {
		if first[i] != second[i] {
			return false
		}
	}
	return true
}
