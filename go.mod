module histrio/rssbook

go 1.17

require (
	github.com/gosimple/slug v1.11.0 // indirect
	github.com/gosimple/unidecode v1.0.0 // indirect
    github.com/histrio/rssbook/pkg/audio v0.0.0
    github.com/histrio/rssbook/pkg/loggers v0.0.0
    github.com/histrio/rssbook/pkg/rss v0.0.0
    github.com/histrio/rssbook/pkg/utils v0.0.0
    github.com/histrio/rssbook/pkg/version v0.0.0
)

replace github.com/histrio/rssbook/pkg/audio v0.0.0 => ./pkg/audio
replace github.com/histrio/rssbook/pkg/loggers v0.0.0 => ./pkg/loggers
replace github.com/histrio/rssbook/pkg/rss v0.0.0 => ./pkg/rss
replace github.com/histrio/rssbook/pkg/utils v0.0.0 => ./pkg/utils
replace github.com/histrio/rssbook/pkg/version v0.0.0 => ./pkg/version
