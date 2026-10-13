package factory

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseCopyBlock(t *testing.T) {
	out := `--
-- PostgreSQL database dump
--

SET statement_timeout = 0;

COPY public.observations_station (id, name, lat) FROM stdin;
1	Station A	12.3
2	Station B	\N
\.
`

	block, err := parseCopyBlock(out)
	require.NoError(t, err)
	require.Equal(t, "id, name, lat", block.Columns)
	require.Equal(t, "1\tStation A\t12.3\n2\tStation B\t\\N\n", block.Data)
}

func TestParseCopyBlockError(t *testing.T) {
	_, err := parseCopyBlock("no copy block here")
	require.Error(t, err)
}
