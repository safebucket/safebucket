require 'fluent/plugin/output'
require 'nats/io/client'
require 'json'

module Fluent::Plugin
  class NatsOutput < Fluent::Plugin::Output
    Fluent::Plugin.register_output('nats', self)

    helpers :compat_parameters

    config_param :host, :string, default: 'localhost'
    config_param :port, :integer, default: 4222
    config_param :subject, :string

    def configure(conf)
      compat_parameters_convert(conf, :buffer)
      super
    end

    def start
      super
      @nats = NATS::IO::Client.new
      @nats.connect(servers: ["nats://#{@host}:#{@port}"])
    end

    def shutdown
      @nats.close if @nats
      super
    end

    def write(chunk)
      chunk.each do |time, record|
        @nats.publish(@subject, record.to_json)
      end
    end
  end
end
