module github.com/openmeterio/openmeter

go 1.26.0

// ee: https://github.com/oklog/run/pull/35
replace github.com/oklog/run => github.com/openmeterio/run v0.0.0-20250217124527-c72029d4b634

require (
	cirello.io/pglock v1.16.1
	entgo.io/ent v0.14.6
	github.com/AppsFlyer/go-sundheit v0.6.0
	github.com/ClickHouse/clickhouse-go/v2 v2.46.0
	github.com/IBM/sarama v1.48.2
	github.com/ThreeDotsLabs/watermill v1.5.2
	github.com/ThreeDotsLabs/watermill-kafka/v3 v3.1.2
	github.com/XSAM/otelsql v0.42.0
	github.com/alpacahq/alpacadecimal v0.0.9
	github.com/avast/retry-go/v4 v4.7.0
	github.com/bhmj/jsonslice v1.1.3
	github.com/brianvoe/gofakeit/v6 v6.28.0
	github.com/brunoga/deep v1.3.1
	github.com/cloudevents/sdk-go/v2 v2.16.2
	github.com/confluentinc/confluent-kafka-go/v2 v2.14.1
	github.com/forscht/namegen v1.0.1
	github.com/getkin/kin-openapi v0.138.0
	github.com/go-chi/chi/v5 v5.2.5
	github.com/go-chi/cors v1.2.2
	github.com/go-chi/render v1.0.3
	github.com/go-co-op/gocron/v2 v2.21.2
	github.com/go-slog/otelslog v0.3.0
	github.com/go-viper/mapstructure/v2 v2.5.0
	github.com/golang-cz/devslog v0.0.15
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/golang-migrate/migrate/v4 v4.19.1
	github.com/google/uuid v1.6.0
	github.com/google/wire v0.7.0
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/huandu/go-sqlbuilder v1.41.0
	github.com/invopop/gobl v0.403.0
	github.com/jackc/pgx/v5 v5.9.2
	github.com/lmittmann/tint v1.1.3
	github.com/mitchellh/mapstructure v1.5.0
	github.com/oapi-codegen/nethttp-middleware v1.1.2
	github.com/oapi-codegen/nullable v1.1.0
	github.com/oapi-codegen/runtime v1.4.1
	github.com/oklog/run v1.1.1-0.20240127200640-eee6e044b77c
	github.com/oklog/ulid/v2 v2.1.1
	github.com/oliveagle/jsonpath v0.1.4
	github.com/peterbourgon/ctxdata/v4 v4.0.0
	github.com/peterldowns/pgtestdb v0.1.1
	github.com/prometheus/client_golang v1.23.2
	github.com/prometheus/client_model v0.6.2
	github.com/prometheus/common v0.67.5
	github.com/qmuntal/stateless v1.8.0
	github.com/rcrowley/go-metrics v0.0.0-20250401214520-65e299d6c5c9
	github.com/redis/go-redis/extra/redisotel/v9 v9.19.0
	github.com/redis/go-redis/v9 v9.19.0
	github.com/rickb777/period v1.0.27
	github.com/sagikazarmark/mapstructurex v0.1.0
	github.com/samber/lo v1.53.0
	github.com/samber/mo v1.16.0
	github.com/samber/slog-multi v1.8.0
	github.com/spf13/cobra v1.10.2
	github.com/spf13/pflag v1.0.10
	github.com/spf13/viper v1.21.0
	github.com/stretchr/testify v1.11.1
	github.com/stripe/stripe-go/v80 v80.2.1
	github.com/svix/svix-webhooks v1.94.0
	github.com/wI2L/jsondiff v0.7.1
	go.opentelemetry.io/contrib/bridges/otelslog v0.18.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.68.0
	go.opentelemetry.io/contrib/instrumentation/runtime v0.68.0
	go.opentelemetry.io/otel v1.43.0
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc v0.19.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.43.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.43.0
	go.opentelemetry.io/otel/exporters/prometheus v0.65.0
	go.opentelemetry.io/otel/log v0.19.0
	go.opentelemetry.io/otel/metric v1.43.0
	go.opentelemetry.io/otel/sdk v1.43.0
	go.opentelemetry.io/otel/sdk/log v0.19.0
	go.opentelemetry.io/otel/sdk/metric v1.43.0
	go.opentelemetry.io/otel/trace v1.43.0
	go.opentelemetry.io/proto/otlp v1.10.0 // indirect
	golang.org/x/exp v0.0.0-20260410095643-746e56fc9e2f
	google.golang.org/grpc v1.81.1
	google.golang.org/protobuf v1.36.12-0.20260120151049-f2248ac996af
)

require (
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/awalterschulze/goderive v0.5.1 // indirect
	github.com/bhmj/xpression v0.9.4 // indirect
	github.com/bitfield/gotestdox v0.2.2 // indirect
	github.com/bmatcuk/doublestar v1.3.4 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/containerd/console v1.0.5 // indirect
	github.com/containerd/containerd v1.7.32 // indirect
	github.com/containerd/containerd/api v1.9.0 // indirect
	github.com/containerd/continuity v0.4.5 // indirect
	github.com/containerd/platforms v1.0.0-rc.1 // indirect
	github.com/dave/jennifer v1.7.1 // indirect
	github.com/dnephin/pflag v1.0.7 // indirect
	github.com/docker/cli v29.2.0+incompatible // indirect
	github.com/docker/go-connections v0.7.0 // indirect
	github.com/expr-lang/expr v1.17.8 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/go-openapi/swag/jsonname v0.25.4 // indirect
	github.com/gofrs/flock v0.12.1 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/huandu/go-clone v1.7.3 // indirect
	github.com/jmattheis/goverter v1.9.3 // indirect
	github.com/jonboulle/clockwork v0.5.0 // indirect
	github.com/kisielk/gotool v1.0.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20250317134145-8bc96cf8fc35 // indirect
	github.com/moby/moby/api v1.54.2 // indirect
	github.com/moby/moby/client v0.4.1 // indirect
	github.com/moby/sys/atomicwriter v0.1.0 // indirect
	github.com/moby/sys/capability v0.4.0 // indirect
	github.com/moby/term v0.5.2 // indirect
	github.com/oapi-codegen/oapi-codegen/v2 v2.6.1-0.20260403235458-a76544bd16ff // indirect
	github.com/oasdiff/yaml v0.0.9 // indirect
	github.com/oasdiff/yaml3 v0.0.12 // indirect
	github.com/onsi/ginkgo v1.16.5 // indirect
	github.com/onsi/gomega v1.38.2 // indirect
	github.com/pb33f/ordered-map/v2 v2.3.1 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/prometheus/otlptranslator v1.0.0 // indirect
	github.com/samber/slog-common v0.21.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.2 // indirect
	github.com/shirou/gopsutil/v4 v4.25.7 // indirect
	github.com/speakeasy-api/jsonpath v0.6.0 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/woodsbury/decimal128 v1.4.0 // indirect
	github.com/zclconf/go-cty-yaml v1.1.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.43.0 // indirect
	go.yaml.in/yaml/v2 v2.4.4 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	go.yaml.in/yaml/v4 v4.0.0-rc.2 // indirect
	golang.org/x/net v0.54.0 // indirect
	gotest.tools/gotestsum v1.13.0 // indirect
	k8s.io/api v0.36.1 // indirect
	k8s.io/apimachinery v0.36.1 // indirect
	k8s.io/client-go v0.36.1 // indirect
)

require (
	ariga.io/atlas v0.36.2-0.20250730182955-2c6300d0a3e1 // indirect
	cloud.google.com/go v0.123.0 // indirect
	github.com/ClickHouse/ch-go v0.71.0
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/ajg/form v1.5.1 // indirect
	github.com/andybalholm/brotli v1.2.0 // indirect
	github.com/apapsch/go-jsonmerge/v2 v2.0.0 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/aws/aws-sdk-go-v2 v1.41.5 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.31.12 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.21 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.21 // indirect
	github.com/aws/smithy-go v1.24.2 // indirect
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/buger/jsonparser v1.1.2 // indirect
	github.com/cespare/xxhash/v2 v2.3.0
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dlclark/regexp2 v1.11.5 // indirect
	github.com/dnwe/otelsarama v0.0.0-20240308230250-9388d9d40bc0 // indirect
	github.com/dprotaso/go-yit v0.0.0-20220510233725-9ba8df137936 // indirect
	github.com/eapache/go-resiliency v1.7.0 // indirect
	github.com/eapache/queue v1.1.0 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-faster/city v1.0.1 // indirect
	github.com/go-faster/errors v0.7.1 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/inflect v0.21.5
	github.com/go-openapi/jsonpointer v0.22.4 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/subcommands v1.2.0 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/gosimple/unidecode v1.0.1
	github.com/govalues/decimal v0.1.36
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.28.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hashicorp/hcl/v2 v2.23.0 // indirect
	github.com/huandu/xstrings v1.4.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/invopop/jsonschema v0.14.0 // indirect
	github.com/invopop/yaml v0.3.1
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jcmturner/aescts/v2 v2.0.0 // indirect
	github.com/jcmturner/dnsutils/v2 v2.0.0 // indirect
	github.com/jcmturner/gofork v1.7.6 // indirect
	github.com/jcmturner/gokrb5/v8 v8.4.4 // indirect
	github.com/jcmturner/rpc/v2 v2.0.3 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12
	github.com/klauspost/compress v1.18.6 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/lib/pq v1.12.3 // indirect
	github.com/lithammer/shortuuid/v3 v3.0.7 // indirect
	github.com/mailru/easyjson v0.9.1 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/paulmach/orb v0.12.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/perimeterx/marshmallow v1.1.5 // indirect
	github.com/pierrec/lz4/v4 v4.1.26 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/procfs v0.20.1 // indirect
	github.com/redis/go-redis/extra/rediscmd/v9 v9.19.0 // indirect
	github.com/rickb777/plural v1.4.10 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/sergi/go-diff v1.4.0 // indirect
	github.com/shopspring/decimal v1.4.0
	github.com/sirupsen/logrus v1.9.4 // indirect
	github.com/sony/gobreaker v1.0.0 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/speakeasy-api/openapi-overlay v0.10.2 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/vmware-labs/yaml-jsonpath v0.3.2 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.2.0
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/zclconf/go-cty v1.16.2 // indirect
	github.com/zeebo/xxh3 v1.1.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.67.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutlog v0.19.0
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/crypto v0.51.0 // indirect
	golang.org/x/mod v0.35.0 // indirect
	golang.org/x/sync v0.20.0
	golang.org/x/sys v0.44.0 // indirect
	golang.org/x/term v0.43.0 // indirect
	golang.org/x/text v0.37.0
	golang.org/x/time v0.15.0 // indirect
	golang.org/x/tools v0.44.0 // indirect
	google.golang.org/genproto v0.0.0-20260217215200-42d3e9bedb6d // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260401024825-9d38bb4040a9 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260401024825-9d38bb4040a9 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

tool (
	github.com/awalterschulze/goderive
	github.com/google/wire/cmd/wire
	github.com/jmattheis/goverter/cmd/goverter
	github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen
	gotest.tools/gotestsum
)
