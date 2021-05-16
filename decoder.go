package fig

type Decoder string

const (
	DecoderYaml Decoder = Decoder(".yaml")
	DecoderYml          = Decoder(".yml")
	DecoderJSON         = Decoder(".json")
	DecoderToml         = Decoder(".toml")
)
