
cluster package
===============

**Discard!**


    目前这个实现方案要考虑的东西挺多的。分布式方式暂时采用nsq来同步了。
    
缺点:
  1. 需要依赖一个服务
  2. nsq并不保证消息的顺序严格一致