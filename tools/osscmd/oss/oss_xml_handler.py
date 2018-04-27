#!/usr/bin/env python
#coding=utf-8
from xml.dom import minidom

def get_tag_text(element, tag):
    nodes = element.getElementsByTagName(tag)
    if len(nodes) == 0:
        return ""
    else:
        node = nodes[0]
    rc = ""
    for node in node.childNodes:
        if node.nodeType in ( node.TEXT_NODE, node.CDATA_SECTION_NODE):
            rc = rc + node.data
    if rc == "true":
        return True
    elif rc == "false":
        return False
    return rc

class ErrorXml:
    def __init__(self, xml_string):
        self.xml = minidom.parseString(xml_string)
        self.code = get_tag_text(self.xml, 'Code')
        self.msg = get_tag_text(self.xml, 'Message')
        self.resource = get_tag_text(self.xml, 'Resource')
        self.request_id = get_tag_text(self.xml, 'RequestId')
        self.host_id = get_tag_text(self.xml, 'HostId')
    
    def show(self):
        print "Code: %s\nMessage: %s\nResource: %s\nRequestId: %s \nHostId: %s" % (self.code, self.msg, self.resource, self.request_id, self.host_id)

class Owner:
    def __init__(self, xml_element):
        self.element = xml_element
        self.id = get_tag_text(self.element, "ID")
        self.display_name = get_tag_text(self.element, "DisplayName")
    
    def show(self):
        print "ID: %s\nDisplayName: %s" % (self.id, self.display_name)

class Bucket:
    def __init__(self, xml_element):
        self.element = xml_element
        self.location = get_tag_text(self.element, "Location")
        self.name = get_tag_text(self.element, "Name")
        self.creation_date = get_tag_text(self.element, "CreationDate")
    
    def show(self):
        print "Name: %s\nCreationDate: %s\nLocation: %s" % (self.name, self.creation_date, self.location)

class GetServiceXml:
    def __init__(self, xml_string):
        self.xml = minidom.parseString(xml_string)
        self.owner = Owner(self.xml.getElementsByTagName('Owner')[0])
        self.buckets = self.xml.getElementsByTagName('Bucket')
        self.bucket_list = []
        for b in self.buckets:
            self.bucket_list.append(Bucket(b))

    def show(self):
        print "Owner:"
        self.owner.show()
        print "\nBucket list:"
        for b in self.bucket_list:
            b.show()
            print ""

    def list(self):
        bl = []
        for b in self.bucket_list:
            bl.append((b.name, b.creation_date, b.location))
        return bl
    
class Content:
    def __init__(self, xml_element):
        self.element = xml_element
        self.key = get_tag_text(self.element, "Key")        
        self.last_modified = get_tag_text(self.element, "LastModified")        
        self.etag = get_tag_text(self.element, "ETag")        
        self.size = get_tag_text(self.element, "Size")        
        self.owner = Owner(self.element.getElementsByTagName('Owner')[0])
        self.storage_class = get_tag_text(self.element, "StorageClass")        

    def show(self):
        print "Key: %s\nLastModified: %s\nETag: %s\nSize: %s\nStorageClass: %s" % (self.key, self.last_modified, self.etag, self.size, self.storage_class)
        self.owner.show()

class Part:
    def __init__(self, xml_element):
        self.element = xml_element
        self.part_num = get_tag_text(self.element, "PartNumber")        
        self.object_name = get_tag_text(self.element, "PartName")
        self.object_size = get_tag_text(self.element, "PartSize")
        self.etag = get_tag_text(self.element, "ETag")

    def show(self):
        print "PartNumber: %s\nPartName: %s\nPartSize: %s\nETag: %s\n" % (self.part_num, self.object_name, self.object_size, self.etag)

class PostObjectGroupXml:
    def __init__(self, xml_string):
        self.xml = minidom.parseString(xml_string)
        self.bucket = get_tag_text(self.xml, 'Bucket')
        self.key = get_tag_text(self.xml, 'Key')
        self.size = get_tag_text(self.xml, 'Size')
        self.etag = get_tag_text(self.xml, "ETag")

    def show(self):
        print "Post Object Group, Bucket: %s\nKey: %s\nSize: %s\nETag: %s" % (self.bucket, self.key, self.size, self.etag)

class GetObjectGroupIndexXml:
    def __init__(self, xml_string):
        self.xml = minidom.parseString(xml_string)
        self.bucket = get_tag_text(self.xml, 'Bucket')
        self.key = get_tag_text(self.xml, 'Key')
        self.etag = get_tag_text(self.xml, 'Etag')
        self.file_length = get_tag_text(self.xml, 'FileLength')
        self.index_list = []
        index_lists = self.xml.getElementsByTagName('Part')
        for i in index_lists:
            self.index_list.append(Part(i))

    def list(self):
        index_list = []
        for i in self.index_list:
            index_list.append((i.part_num, i.object_name, i.object_size, i.etag))
        return index_list

    def show(self):
        print "Bucket: %s\nObject: %s\nEtag: %s\nObjectSize: %s" % (self.bucket, self.key, self.etag, self.file_length)
        print "\nPart list:"
        for p in self.index_list:
            p.show()

class GetObjectLinkIndexXml:
    def __init__(self, xml_string):
        self.xml = minidom.parseString(xml_string)
        self.index_list = []
        index_lists = self.xml.getElementsByTagName('Part')
        for i in index_lists:
            self.index_list.append(Part(i))

    def list(self):
        index_list = []
        for i in self.index_list:
            index_list.append((i.part_num, i.object_name))
        return index_list

    def show(self):
        print "\nPart list:"
        for p in self.index_list:
            p.show()

class GetObjectLinkInfoXml:
    def __init__(self, xml_string):
        self.xml = minidom.parseString(xml_string)
        self.bucket = get_tag_text(self.xml, 'Bucket')
        self.type = get_tag_text(self.xml, 'Type')
        self.key = get_tag_text(self.xml, 'Key')
        self.etag = get_tag_text(self.xml, 'ETag')
        self.last_modified = get_tag_text(self.xml, 'LastModified')
        self.index_list = []
        index_lists = self.xml.getElementsByTagName('Part')
        for i in index_lists:
            self.index_list.append(Part(i))

    def list(self):
        index_list = []
        for i in self.index_list:
            index_list.append((i.part_num, i.object_name, i.object_size, i.etag))
        return index_list

    def show(self):
        print "Bucket: %s\nType: %s\nObject: %s\nEtag: %s\nLastModified: %s" % (self.bucket, self.type, self.key, self.etag, self.last_modified)
        print "\nPart list:"
        for p in self.index_list:
            p.show()

class GetBucketXml:
    def __init__(self, xml_string):
        self.xml = minidom.parseString(xml_string)
        self.name = get_tag_text(self.xml, 'Name')
        self.prefix = get_tag_text(self.xml, 'Prefix')
        self.marker = get_tag_text(self.xml, 'Marker')
        self.nextmarker = get_tag_text(self.xml, 'NextMarker')
        self.maxkeys = get_tag_text(self.xml, 'MaxKeys')
        self.delimiter = get_tag_text(self.xml, 'Delimiter')
        self.is_truncated = get_tag_text(self.xml, 'IsTruncated')

        self.prefix_list = []
        prefixes = self.xml.getElementsByTagName('CommonPrefixes')
        for p in prefixes:
            tag_txt = get_tag_text(p, "Prefix")
            self.prefix_list.append(tag_txt)

        self.content_list = []
        contents = self.xml.getElementsByTagName('Contents')
        for c in contents:
            self.content_list.append(Content(c))

    def show(self):
        print "Name: %s\nPrefix: %s\nMarker: %s\nNextMarker: %s\nMaxKeys: %s\nDelimiter: %s\nIsTruncated: %s" % (self.name, self.prefix, self.marker, self.nextmarker, self.maxkeys, self.delimiter, self.is_truncated)
        print "\nPrefix list:"
        for p in self.prefix_list:
            print p
        print "\nContent list:"
        for c in self.content_list:
            c.show()
            print ""

    def list(self):
        cl = []
        pl = []
        for c in self.content_list:
            cl.append((c.key, c.last_modified, c.etag, c.size, c.owner.id, c.owner.display_name, c.storage_class))
        for p in self.prefix_list:
            pl.append(p)

        return (cl, pl)
 
class GetBucketAclXml:
    def __init__(self, xml_string):
        self.xml = minidom.parseString(xml_string)
        if len(self.xml.getElementsByTagName('Owner')) != 0:
            self.owner = Owner(self.xml.getElementsByTagName('Owner')[0])
        else:
            self.owner = "" 
        self.grant = get_tag_text(self.xml, 'Grant')

    def show(self):
        print "Owner Name: %s\nOwner ID: %s\nGrant: %s" % (self.owner.id, self.owner.display_name, self.grant)
 
class GetBucketLocationXml:
    def __init__(self, xml_string):
        self.xml = minidom.parseString(xml_string)
        self.location = get_tag_text(self.xml, 'LocationConstraint')
    
    def show(self):
        print "LocationConstraint: %s" % (self.location)

class GetInitUploadIdXml:
    def __init__(self, xml_string):
        self.xml = minidom.parseString(xml_string)
        self.bucket = get_tag_text(self.xml, 'Bucket')
        self.object = get_tag_text(self.xml, 'Key')
        self.key = get_tag_text(self.xml, 'Key')
        self.upload_id = get_tag_text(self.xml, 'UploadId')
        self.marker = get_tag_text(self.xml, 'Marker')
       
    def show(self):
        print " "     

class Upload:
    def __init__(self, xml_element):
        self.element = xml_element
        self.key = get_tag_text(self.element, "Key")        
        self.upload_id = get_tag_text(self.element, "UploadId")

class GetMultipartUploadsXml:
    def __init__(self, xml_string):
        self.xml = minidom.parseString(xml_string)
        self.bucket = get_tag_text(self.xml, 'Bucket')
        self.key_marker = get_tag_text(self.xml, 'KeyMarker')
        self.upload_id_marker = get_tag_text(self.xml, 'UploadIdMarker')
        self.next_key_marker = get_tag_text(self.xml, 'NextKeyMarker')
        self.next_upload_id_marker = get_tag_text(self.xml, 'NextUploadIdMarker')
        self.delimiter = get_tag_text(self.xml, 'Delimiter')
        self.prefix = get_tag_text(self.xml, 'Prefix')
        self.max_uploads = get_tag_text(self.xml, 'MaxUploads')
        self.is_truncated = get_tag_text(self.xml, 'IsTruncated')

        self.prefix_list = []
        prefixes = self.xml.getElementsByTagName('CommonPrefixes')
        for p in prefixes:
            tag_txt = get_tag_text(p, "Prefix")
            self.prefix_list.append(tag_txt)

        self.content_list = []
        contents = self.xml.getElementsByTagName('Upload')
        for c in contents:
            self.content_list.append(Upload(c))

    def list(self):
        cl = []
        pl = []
        for c in self.content_list:
            cl.append((c.key, c.upload_id))
        for p in self.prefix_list:
            pl.append(p)

        return (cl, pl)

class MultiPart:
    def __init__(self, xml_element):
        self.element = xml_element
        self.part_number = get_tag_text(self.element, 'PartNumber')
        self.last_modified = get_tag_text(self.element, 'LastModified')
        self.etag = get_tag_text(self.element, 'ETag')
        self.size = get_tag_text(self.element, 'Size')

class GetPartsXml:
    def __init__(self, xml_string):
        self.xml = minidom.parseString(xml_string)
        self.bucket = get_tag_text(self.xml, 'Bucket')
        self.key = get_tag_text(self.xml, 'Key')
        self.upload_id = get_tag_text(self.xml, 'UploadId')
        self.storage_class = get_tag_text(self.xml, 'StorageClass')
        self.next_part_number_marker = get_tag_text(self.xml, 'NextPartNumberMarker')
        self.max_parts = get_tag_text(self.xml, 'MaxParts')
        self.is_truncated = get_tag_text(self.xml, 'IsTruncated')
        self.part_number_marker = get_tag_text(self.xml, 'PartNumberMarker')
        
        self.content_list = []
        contents = self.xml.getElementsByTagName('Part')
        for c in contents:
            self.content_list.append(MultiPart(c))

    def list(self):
        cl = []
        for c in self.content_list:
            cl.append((c.part_number, c.etag, c.size, c.last_modified))
        return cl

class CompleteUploadXml:
    def __init__(self, xml_string):
        self.xml = minidom.parseString(xml_string)
        self.location = get_tag_text(self.xml, 'Location')
        self.bucket = get_tag_text(self.xml, 'Bucket')
        self.key = get_tag_text(self.xml, 'Key')
        self.etag = get_tag_text(self.xml, "ETag")

class DeletedObjectsXml:
    def __init__(self, xml_string):
        self.xml = minidom.parseString(xml_string)
        contents = self.xml.getElementsByTagName('Deleted')
        self.content_list = []
        for c in contents:
            self.content_list.append(get_tag_text(c, 'Key'))
    def list(self):
        cl = []
        for c in self.content_list:
            cl.append(c)
        return cl

class CnameInfoPart:
    def __init__(self, xml_element):
        self.element = xml_element
        self.cname = get_tag_text(self.element, 'Cname')
        self.bucket = get_tag_text(self.element, 'Bucket')
        self.status = get_tag_text(self.element, 'Status')
        self.lastmodifytime = get_tag_text(self.element, 'LastModifyTime')

class CnameToBucketXml:
    def __init__(self, xml_string):
        self.xml = minidom.parseString(xml_string)
        self.content_list = []
        contents = self.xml.getElementsByTagName('CnameInfo')
        for c in contents:
            self.content_list.append(CnameInfoPart(c))

    def list(self):
        cl = []
        for c in self.content_list:
            cl.append((c.cname, c.bucket, c.status, c.lastmodifytime))
        return cl

class RedirectXml:
    def __init__(self, xml_string):
        self.xml = minidom.parseString(xml_string)
        self.endpoint = get_tag_text(self.xml, 'Endpoint')
    def Endpoint(self):
        return self.endpoint

if __name__ == "__main__":
    pass
