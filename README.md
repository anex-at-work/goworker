# goworker

This is a fork of [goworker](https://github.com/benmanns/goworker), you can find full information at project page.

## Scheduling

This fork extends original functionality through additional function EnqueueAt.

And now you can push your jobs at the scheduled time:

```go
goworker.EnqueueAt(&goworker.JobAt{
  Queue: "myqueue",
  Payload: goworker.Payload{
      Class: "MyClass",
      Args: []interface{}{"hi", "there"},
  },
  RunAt: time.Now().Add(time.Minute * 2) // 2 minute later
})
```

If you use Resque to enqueued tasks, you need to extend Resque by next code:

```ruby
module Resque
  class << self
    def enqueue_at time, klass, *args
      time = Time.zone.now + time if time.is_a? ActiveSupport::Duration
      Resque.redis.zadd 'zqueue:' + queue_from_class(klass).to_s,
        time.to_f, Resque.encode(class: klass.name, args: args, run_at: time.to_f)
    end
  end
end
```

and then use:

```ruby
Resque.enqueue_at 2.minutes, MyClass, ['hi', 'there'] # 2 minute later
# or
Resque.enqueue_at 2.minutes.from_now, MyClass, ['hi', 'there'] # the same, but sends exactly time
```

### Remark

This fork uses Redis sorted set (ZADD, ZREMRANGEBYSCORE etc.) and modifies default name of Resque queue: from

```
resque:queue
```

to

```
resque:zqueue
```

(with "z"). Don't forget about this!

## Contributing

1. [Fork it](https://github.com/benmanns/goworker/fork)
2. Create your feature branch (`git checkout -b my-new-feature`)
3. Commit your changes (`git commit -am 'Add some feature'`)
4. Push to the branch (`git push origin my-new-feature`)
5. Create new Pull Request
