package common

import (
	"fmt"
	"strconv"

	"github.com/BenasB/bx2cloud/internal/cli/exits"
)

func ParseUint32Arg(args *[]string) (uint32, exits.ExitCode, error) {
	if len(*args) == 0 {
		return 0, exits.MISSING_ARGUMENT, fmt.Errorf("missing argument")
	}

	argString := (*args)[0]

	arg, err := strconv.ParseUint(argString, 10, 32)
	if err != nil {
		return 0, exits.BAD_ARGUMENT, fmt.Errorf("Could not convert '%d' to an unsigned integer\n", arg)
	}

	*args = (*args)[1:]

	return uint32(arg), exits.SUCCESS, nil
}
