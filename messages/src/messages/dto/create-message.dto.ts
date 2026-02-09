export class CreateMessageDto {
  conversationId?: string;
  senderId: string;
  recipientId?: string;
  content: string;
  sender?: string;
  timestamp?: Date;
  isRead?: boolean;
  isEncrypted?: boolean;
  ephemeralPublicKey?: string;
}