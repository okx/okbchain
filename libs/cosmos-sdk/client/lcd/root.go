package lcd

import (
	"fmt"
	"github.com/gogo/gateway"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	grpctypes "github.com/okx/okbchain/libs/cosmos-sdk/types/grpc"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/okx/okbchain/libs/tendermint/libs/log"
	"github.com/okx/okbchain/libs/tendermint/node"
	"github.com/okx/okbchain/libs/tendermint/rpc/client/local"
	tmrpcserver "github.com/okx/okbchain/libs/tendermint/rpc/jsonrpc/server"
	"github.com/rakyll/statik/fs"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/okx/okbchain/libs/cosmos-sdk/client/context"
	"github.com/okx/okbchain/libs/cosmos-sdk/client/flags"
	"github.com/okx/okbchain/libs/cosmos-sdk/codec"
	keybase "github.com/okx/okbchain/libs/cosmos-sdk/crypto/keys"

	// unnamed import of statik for swagger UI support
	_ "github.com/okx/okbchain/libs/cosmos-sdk/client/lcd/statik"
)

// RestServer represents the Light Client Rest server
type RestServer struct {
	Mux     *mux.Router
	CliCtx  context.CLIContext
	KeyBase keybase.Keybase
	Cdc     *codec.CodecProxy

	log      log.Logger
	listener net.Listener

	GRPCGatewayRouter *runtime.ServeMux
}

// CustomGRPCHeaderMatcher for mapping request headers to
// GRPC metadata.
// HTTP headers that start with 'Grpc-Metadata-' are automatically mapped to
// gRPC metadata after removing prefix 'Grpc-Metadata-'. We can use this
// CustomGRPCHeaderMatcher if headers don't start with `Grpc-Metadata-`
func CustomGRPCHeaderMatcher(key string) (string, bool) {
	switch strings.ToLower(key) {
	case grpctypes.GRPCBlockHeightHeader:
		return grpctypes.GRPCBlockHeightHeader, true
	default:
		return runtime.DefaultHeaderMatcher(key)
	}
}

// NewRestServer creates a new rest server instance
func NewRestServer(cdc *codec.CodecProxy, interfaceReg jsonpb.AnyResolver, tmNode *node.Node) *RestServer {
	rootRouter := mux.NewRouter()
	cliCtx := context.NewCLIContext().WithProxy(cdc)
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout)).With("module", "rest-server")
	if tmNode != nil {
		cliCtx = cliCtx.WithChainID(tmNode.ConsensusState().GetState().ChainID)
		cliCtx.Client = local.New(tmNode)
		logger = tmNode.Logger.With("module", "rest-server")
	} else {
		cliCtx = cliCtx.WithChainID(viper.GetString(flags.FlagChainID))
	}

	cliCtx.TrustNode = true

	marshalerOption := NewJSONMarshalAdapter(&gateway.JSONPb{
		EmitDefaults: true,
		Indent:       "  ",
		OrigName:     true,
		AnyResolver:  interfaceReg,
	}, cdc)

	return &RestServer{
		Mux:    rootRouter,
		CliCtx: cliCtx,
		Cdc:    cdc,

		log: logger,
		GRPCGatewayRouter: runtime.NewServeMux(
			// Custom marshaler option is required for gogo proto
			runtime.WithMarshalerOption(runtime.MIMEWildcard, marshalerOption),

			// This is necessary to get error details properly
			// marshalled in unary requests.
			runtime.WithProtoErrorHandler(runtime.DefaultHTTPProtoErrorHandler),

			// Custom header matcher for mapping request headers to
			// GRPC metadata
			runtime.WithIncomingHeaderMatcher(CustomGRPCHeaderMatcher),
		),
	}
}

func (rs *RestServer) Logger() log.Logger {
	return rs.log
}

// Start starts the rest server
func (rs *RestServer) Start(listenAddr string, maxOpen int, readTimeout, writeTimeout uint, maxBodyBytes int64, cors bool) (err error) {
	//trapSignal(func() {
	//	err := rs.listener.Close()
	//	rs.log.Error("error closing listener", "err", err)
	//})

	cfg := tmrpcserver.DefaultConfig()
	cfg.MaxOpenConnections = maxOpen
	cfg.ReadTimeout = time.Duration(readTimeout) * time.Second
	cfg.WriteTimeout = time.Duration(writeTimeout) * time.Second
	if maxBodyBytes > 0 {
		cfg.MaxBodyBytes = maxBodyBytes
	}

	rs.listener, err = tmrpcserver.Listen(listenAddr, cfg)
	if err != nil {
		return
	}

	rs.registerGRPCGatewayRoutes()

	rs.log.Info(
		fmt.Sprintf(
			"Starting application REST service (chain-id: %q)...",
			viper.GetString(flags.FlagChainID),
		),
	)

	var h http.Handler = rs.Mux
	if cors {
		allowAllCORS := handlers.CORS(handlers.AllowedHeaders([]string{"Content-Type"}))
		h = allowAllCORS(h)
	}

	return tmrpcserver.Serve(rs.listener, h, rs.log, cfg)
}

func (s *RestServer) registerGRPCGatewayRoutes() {
	s.Mux.PathPrefix("/").Handler(s.GRPCGatewayRouter)
}

// ServeCommand will start the application REST service as a blocking process. It
// takes a codec to create a RestServer object and a function to register all
// necessary routes.
func ServeCommand(cdc *codec.CodecProxy, interfaceReg jsonpb.AnyResolver, registerRoutesFn func(*RestServer)) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rest-server",
		Short: "Start LCD (light-client daemon), a local REST server",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			rs := NewRestServer(cdc, interfaceReg, nil)

			registerRoutesFn(rs)
			//rs.registerSwaggerUI()

			// Start the rest server and return error if one exists
			err = rs.Start(
				viper.GetString(flags.FlagListenAddr),
				viper.GetInt(flags.FlagMaxOpenConnections),
				uint(viper.GetInt(flags.FlagRPCReadTimeout)),
				uint(viper.GetInt(flags.FlagRPCWriteTimeout)),
				viper.GetInt64(flags.FlagMaxBodyBytes),
				viper.GetBool(flags.FlagUnsafeCORS),
			)

			return err
		},
	}

	return flags.RegisterRestServerFlags(cmd)
}

func StartRestServer(cdc *codec.CodecProxy, interfaceReg jsonpb.AnyResolver, registerRoutesFn func(*RestServer), tmNode *node.Node, addr string) error {
	rs := NewRestServer(cdc, interfaceReg, tmNode)

	registerRoutesFn(rs)
	//rs.registerSwaggerUI()
	rs.log.Info("start rest server")
	// Start the rest server and return error if one exists
	return rs.Start(
		addr,
		viper.GetInt(flags.FlagMaxOpenConnections),
		uint(viper.GetInt(flags.FlagRPCReadTimeout)),
		uint(viper.GetInt(flags.FlagRPCWriteTimeout)),
		viper.GetInt64(flags.FlagMaxBodyBytes),
		viper.GetBool(flags.FlagUnsafeCORS),
	)
}

func (rs *RestServer) registerSwaggerUI() {
	statikFS, err := fs.New()
	if err != nil {
		panic(err)
	}
	staticServer := http.FileServer(statikFS)
	rs.Mux.PathPrefix("/swagger-ui/").Handler(http.StripPrefix("/swagger-ui/", staticServer))
}
