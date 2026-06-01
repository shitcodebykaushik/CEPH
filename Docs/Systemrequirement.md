# CPU 
- CephFS Metadata Server are cpu intensive .
- They are single threaded and perform best with CPUs with the high clock rate . MDs Server do not need  a large number of cpu core unless they are also hosting other services such as SSD OSFDs for the CephFs metadata-Pool.
- OSD nodes need enough processing power to run the  RADOS service,to calculate .
- 