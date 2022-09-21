import {
  Column,
  CreateDateColumn,
  Entity,
  JoinColumn,
  ManyToOne,
  OneToOne,
  PrimaryGeneratedColumn,
} from 'typeorm';

import { Channel } from './Channel.js';
import { ChannelFollowEvent } from './channelEvents/Follow.js';

export enum EventType {
  FOLLOW = 'follow',
  SUBSCRIPTION = 'subscription',
  RESUBSCRIPTION = 'resubscription',
  DONATION = 'donation',
  HOST = 'host',
  RAID = 'raid',
  MODERATOR_ADD = 'moderator_added',
  MODERATOR_REMOVE = 'moderator_remove',
}

@Entity({
  name: 'channel_events_list',
})
export class ChannelEvent {
  @PrimaryGeneratedColumn('uuid')
  id: string;

  @ManyToOne(() => Channel, (c) => c.events)
  @JoinColumn({ name: 'channelId' })
  channel: Channel;

  @Column()
  channelId: string;

  @Column('enum', { enum: EventType })
  type: EventType;

  @OneToOne('ChannelFollowEvent', 'event')
  follow?: [];

  @CreateDateColumn()
  createdAt: Date;
}
