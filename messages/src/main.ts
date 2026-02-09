import { NestFactory } from '@nestjs/core';
import { Transport, MicroserviceOptions } from '@nestjs/microservices';
import { AppModule } from './app.module';

async function bootstrap() {
  const app = await NestFactory.create(AppModule);

  app.connectMicroservice<MicroserviceOptions>({
    transport: Transport.KAFKA,
    options: {
      client: {
        brokers: [`${process.env.KAFKA_HOST || 'localhost'}:${process.env.KAFKA_PORT || 9092}`],
      },
      consumer: {
        groupId: process.env.KAFKA_GROUP_ID || 'messages-service',
      },
    },
  });

  await app.startAllMicroservices();
  await app.listen(process.env.SERVER_PORT || 3000);
}
bootstrap();