// This file is completely vibecoded and needs to be properly reviewed.

package api

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func (c Controller) runOperation(operation Operation, arguments []string) (int, error) {
	flagSet := flag.NewFlagSet(operation.Command, flag.ContinueOnError)
	flagSet.SetOutput(io.Discard)
	flagSet.Usage = func() {}

	values := make(map[string]*string, len(operation.Parameters))
	for _, parameter := range operation.Parameters {
		values[parameter.Name] = flagSet.String(parameter.Flag, "", parameterUsage(parameter))
	}
	body := new(string)
	if operation.HasBody {
		body = flagSet.String(bodyFlag, "", bodyUsage(operation))
	}
	address := flagSet.String(addressFlag, "", addressFlagUsage)

	parseError := flagSet.Parse(arguments)
	if errors.Is(parseError, flag.ErrHelp) {
		return 0, c.writeOperationHelp(operation)
	}
	if parseError != nil {
		return 0, parseError
	}
	if flagSet.NArg() > 0 {
		return 0, fmt.Errorf("unexpected argument: %s (run \"wade api %s --help\" for usage)", flagSet.Arg(0), operation.Command)
	}

	for _, parameter := range operation.Parameters {
		if parameter.Required && *values[parameter.Name] == "" {
			return 0, fmt.Errorf("missing required flag --%s", parameter.Flag)
		}
	}
	if operation.BodyRequired && *body == "" {
		return 0, fmt.Errorf("missing required flag --%s", bodyFlag)
	}

	requestBody, err := c.readBody(*body)
	if err != nil {
		return 0, err
	}

	if *address == "" {
		*address = c.resolveAddress()
	}
	return c.executeRequest(operation, *address, operationPath(operation, values), requestBody)
}

func (c Controller) readBody(value string) ([]byte, error) {
	switch {
	case value == "":
		return nil, nil
	case value == "-":
		body, err := io.ReadAll(c.stdin)
		if err != nil {
			return nil, fmt.Errorf("reading body from stdin: %w", err)
		}
		return body, nil
	case strings.HasPrefix(value, "@"):
		body, err := os.ReadFile(strings.TrimPrefix(value, "@"))
		if err != nil {
			return nil, fmt.Errorf("reading body file: %w", err)
		}
		return body, nil
	default:
		return []byte(value), nil
	}
}

func operationPath(operation Operation, values map[string]*string) string {
	path := operation.Path
	query := url.Values{}
	for _, parameter := range operation.Parameters {
		value := *values[parameter.Name]
		switch {
		case parameter.In == "path":
			path = strings.ReplaceAll(path, "{"+parameter.Name+"}", url.PathEscape(value))
		case value != "":
			query.Set(parameter.Name, value)
		}
	}

	if len(query) > 0 {
		path += "?" + query.Encode()
	}
	return path
}

func (c Controller) executeRequest(operation Operation, address string, path string, body []byte) (int, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	request, err := http.NewRequest(operation.Method, "http://"+address+path, bodyReader)
	if err != nil {
		return 0, err
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return 0, fmt.Errorf("cannot reach WADE at http://%s: %w\nStart the server with \"wade start\" or pass --%s", address, err, addressFlag)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return 0, fmt.Errorf("reading response from WADE: %w", err)
	}

	requestSucceeded := response.StatusCode >= 200 && response.StatusCode <= 299
	if !requestSucceeded {
		problem := strings.TrimSpace(string(responseBody))
		if problem == "" {
			return 0, fmt.Errorf("HTTP %s", response.Status)
		}
		return 0, fmt.Errorf("HTTP %s\n%s", response.Status, problem)
	}

	if len(responseBody) == 0 {
		return 0, nil
	}
	_, err = c.stdout.Write(responseBody)
	return 0, err
}
