import { NodeSDK } from '@opentelemetry/sdk-node';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http';
import { BatchSpanProcessor } from '@opentelemetry/sdk-trace-base';
import { getNodeAutoInstrumentations } from '@opentelemetry/auto-instrumentations-node';
import { KafkaJsInstrumentation } from '@opentelemetry/instrumentation-kafkajs';
import { resourceFromAttributes } from '@opentelemetry/resources';
import { SEMRESATTRS_SERVICE_NAME } from '@opentelemetry/semantic-conventions';

const otlpEndpoint = process.env.OTLP_ENDPOINT || 'localhost:4318';
const serviceName = process.env.OTEL_SERVICE_NAME || 'messaging';

const traceExporter = new OTLPTraceExporter({
  url: otlpEndpoint.includes('://') ? otlpEndpoint : `http://${otlpEndpoint}/v1/traces`,
});

const sdk = new NodeSDK({
  resource: resourceFromAttributes({
    [SEMRESATTRS_SERVICE_NAME]: serviceName,
  }),
  spanProcessor: new BatchSpanProcessor(traceExporter),
  instrumentations: [
    getNodeAutoInstrumentations({
      '@opentelemetry/instrumentation-http': { enabled: true },
      '@opentelemetry/instrumentation-express': { enabled: true },
      '@opentelemetry/instrumentation-nestjs-core': { enabled: true },
      '@opentelemetry/instrumentation-socket.io': { enabled: true },
      '@opentelemetry/instrumentation-mongoose': { enabled: true },
    }),
    new KafkaJsInstrumentation(),
  ],
});

sdk.start();

process.on('SIGTERM', () => {
  sdk.shutdown().catch((err) => console.error('OTel shutdown error:', err));
});
