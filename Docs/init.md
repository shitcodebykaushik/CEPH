# The future of the storage 
- Every file,block and object is ultimately broken down and stored as object inside `RADOS`(Reliable Autonomic Distributed Object store).
- RADOS is the engine that handles the data replication,ensure coding,fault detection and self-healing across the cluster without single point of failure .
- Traditional storage system rely on centralized lookup table(or metadata server ) to know where the data is physically located ,which created the massive bottleneck.
- CEPH completely bypass the `CRUSH` Controlled Replication Under Scalable Hasing . 
- 
# Concept of the Object storage .
- It doesn't follow the traditional way of storing the data . 
- It can store and manage the data volumes ranging from the terabytes to petabytes and beyond . Including the exabyte-scale deployment that power today largest cloud deployements and data intesive applications . 
- Instead of breaking files into blocks or organizing them in hierarchical folders,onject storage treats each piech of data as discreate addressable unit . Unlike file systems that rely on directory structures or block storage that fragments data, object storage maintains complete data integrity(means data is accurate) within each storage unit. 
- It is ideal for archiving static data,such as compliance records,media lib and backup or any data doesn't require frequent modicfication .

# File storage 
- This organizes and store data inside a folder . Files are named,tagged with the metadata(typically the file name ,file type and when it was created and last updated ),and organized in folders under the hierarchy of directories and sub-dir . 

# Block storage 
- It offets an alternative to file-based storage,one with the improved effieciency and performance . Block storage breaks a file into equally sized chunks of data and stored these data blocks sepearately ,under unique address . You dont need a file-folder structure . Instead you can store the collection of blocks anywhere in the file system .
- To access a file ,a server operating system uses the unique address to pull the blocks back together, assembling them into the file.


- Running ceph require handling multi-daemon architecutres .
