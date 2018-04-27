#!/usr/bin/env python
#coding=utf-8
import urllib
import base64
import hmac
import time
from hashlib import sha1 as sha
import os
import sys
import md5
import StringIO
from threading import Thread
import threading
import ConfigParser
import logging
from logging.handlers import RotatingFileHandler
from xml.sax.saxutils import escape
import socket
try:
    from oss.oss_xml_handler import *
except:
    from oss_xml_handler import *

#LOG_LEVEL can be one of DEBUG INFO ERROR CRITICAL WARNNING
DEBUG = False
LOG_LEVEL = "DEBUG" 
PROVIDER = "OSS"
SELF_DEFINE_HEADER_PREFIX = "x-oss-"
if "AWS" == PROVIDER:
    SELF_DEFINE_HEADER_PREFIX = "x-amz-"

def getlogger(debug=DEBUG, log_level=LOG_LEVEL, log_name="log.txt"):
    if not debug:
        logger = logging.getLogger('oss')
        logger.addHandler(EmptyHandler())
        return logger
    logfile = os.path.join(os.getcwd(), log_name)
    max_log_size = 100*1024*1024 #Bytes
    backup_count = 5
    format = \
    "%(asctime)s %(levelname)-8s[%(filename)s:%(lineno)d(%(funcName)s)] %(message)s"
    hdlr = RotatingFileHandler(logfile,
                                  mode='a',
                                  maxBytes=max_log_size,
                                  backupCount=backup_count)
    formatter = logging.Formatter(format)
    hdlr.setFormatter(formatter)
    logger = logging.getLogger("oss")
    logger.addHandler(hdlr)
    if "DEBUG" == log_level.upper():
        logger.setLevel(logging.DEBUG)
    elif "INFO" == log_level.upper():
        logger.setLevel(logging.INFO)
    elif "WARNING" == log_level.upper():
        logger.setLevel(logging.WARNING)
    elif "ERROR" == log_level.upper():
        logger.setLevel(logging.ERROR)
    elif "CRITICAL" == log_level.upper():
        logger.setLevel(logging.CRITICAL)
    else:
        logger.setLevel(logging.ERROR)
    return logger

class EmptyHandler(logging.Handler):
    def __init__(self):
        self.lock = None
    def emit(self, record):
        pass
    def handle(self, record):
        pass
    def createLock(self):
        self.lock = None 

def helper_get_host_from_resp(res, bucket):
    host = helper_get_host_from_headers(res.getheaders(), bucket)
    if not host:
        xml = res.read()
        host = RedirectXml(xml).Endpoint().strip()
        host = helper_get_host_from_endpoint(host, bucket)
    return host

def helper_get_host_from_headers(headers, bucket):
    mp = convert_header2map(headers)
    location = safe_get_element('location', mp).strip()
    #https://bucket.oss.aliyuncs.com or http://oss.aliyuncs.com/bucket
    location = location.replace("https://", "").replace("http://", "")
    if location.startswith("%s." % bucket):
        location = location[len(bucket)+1:]
    index = location.find('/')
    if index == -1:
        return location
    return location[:index]

def helper_get_host_from_endpoint(host, bucket):
    index = host.find('/')
    if index != -1:
        host = host[:index]
    index = host.find('\\')
    if index != -1:
        host = host[:index]
    index = host.find(bucket)
    if index == 0:
        host = host[len(bucket)+1:]
    return host

def check_bucket_valid(bucket):
    alphabeta = "abcdefghijklmnopqrstuvwxyz0123456789-"
    if len(bucket) < 3 or len(bucket) > 63:
        return False
    if bucket[-1] == "-" or bucket[-1] == "_":
        return False
    if not ((bucket[0] >= 'a' and bucket[0] <= 'z') or (bucket[0] >= '0' and bucket[0] <= '9')):
        return False
    for i in bucket:
        if not i in alphabeta:
            return False
    return True

def check_redirect(res):
    is_redirect = False
    try:
        if res.status == 301 or res.status == 302:
            is_redirect = True
    except:
        pass
    return is_redirect

########## function for Authorization ##########
def _format_header(headers=None):
    '''
    format the headers that self define
    convert the self define headers to lower.
    '''
    if not headers:
        headers = {}
    tmp_headers = {}
    for k in headers.keys():
        if isinstance(headers[k], unicode):
            headers[k] = headers[k].encode('utf-8')

        if k.lower().startswith(SELF_DEFINE_HEADER_PREFIX):
            k_lower = k.lower().strip()
            tmp_headers[k_lower] = headers[k]
        else:
            tmp_headers[k.strip()] = headers[k]
    return tmp_headers

def get_assign(secret_access_key, method, headers=None, resource="/", result=None, debug=DEBUG):
    '''
    Create the authorization for OSS based on header input.
    You should put it into "Authorization" parameter of header.
    '''
    if not headers:
        headers = {}
    if not result:
        result = []
    content_md5 = ""
    content_type = ""
    date = ""
    canonicalized_oss_headers = ""
    logger = getlogger(debug)
    logger.debug("secret_access_key: %s" % secret_access_key)
    content_md5 = safe_get_element('Content-MD5', headers)
    content_type = safe_get_element('Content-Type', headers)
    date = safe_get_element('Date', headers)
    canonicalized_resource = resource
    tmp_headers = _format_header(headers)
    if len(tmp_headers) > 0:
        x_header_list = tmp_headers.keys()
        x_header_list.sort()
        for k in x_header_list:
            if k.startswith(SELF_DEFINE_HEADER_PREFIX):
                canonicalized_oss_headers += "%s:%s\n" % (k, tmp_headers[k]) 
    string_to_sign = method + "\n" + content_md5.strip() + "\n" + content_type + "\n" + date + "\n" + canonicalized_oss_headers + canonicalized_resource
    result.append(string_to_sign)
    logger.debug("method:%s\n content_md5:%s\n content_type:%s\n data:%s\n canonicalized_oss_headers:%s\n canonicalized_resource:%s\n" % (method, content_md5, content_type, date, canonicalized_oss_headers, canonicalized_resource))
    logger.debug("string_to_sign:%s\n \nlength of string_to_sign:%d\n" % (string_to_sign, len(string_to_sign)))
    h = hmac.new(secret_access_key, string_to_sign, sha)
    sign_result = base64.encodestring(h.digest()).strip()
    logger.debug("sign result:%s" % sign_result)
    return sign_result

def get_resource(params=None):
    if not params:
        return ""
    tmp_headers = {}
    for k, v in params.items():
        tmp_k = k.lower().strip()
        tmp_headers[tmp_k] = v
    override_response_list = ['response-content-type', 'response-content-language', \
                              'response-cache-control', 'logging', 'response-content-encoding', \
                              'acl', 'uploadId', 'uploads', 'partNumber', 'group', 'link', \
                              'delete', 'website', 'location', 'objectInfo', \
                              'response-expires', 'response-content-disposition']
    override_response_list.sort()
    resource = ""
    separator = "?"
    for i in override_response_list:
        if tmp_headers.has_key(i.lower()):
            resource += separator
            resource += i
            tmp_key = str(tmp_headers[i.lower()])
            if len(tmp_key) != 0:
                resource += "="
                resource += tmp_key 
            separator = '&'
    return resource

def oss_quote(in_str):
    if not isinstance(in_str, str):
        in_str = str(in_str)
    return urllib.quote(in_str, '')

def append_param(url, params):
    '''
    convert the parameters to query string of URI.
    '''
    l = []
    for k, v in params.items():
        k = k.replace('_', '-')
        if  k == 'maxkeys':
            k = 'max-keys'
        if isinstance(v, unicode):
            v = v.encode('utf-8')
        if v is not None and v != '':
            l.append('%s=%s' % (oss_quote(k), oss_quote(v)))
        elif k == 'acl':
            l.append('%s' % (oss_quote(k)))
        elif v is None or v == '':
            l.append('%s' % (oss_quote(k)))
    if len(l):
        url = url + '?' + '&'.join(l)
    return url

############### Construct XML ###############
def create_object_group_msg_xml(part_msg_list=None):
    '''
    get information from part_msg_list and covert it to xml.
    part_msg_list has special format.
    '''
    if not part_msg_list:
        part_msg_list = []
    xml_string = r'<CreateFileGroup>'
    for part in part_msg_list:
        if len(part) >= 3:
            if isinstance(part[1], unicode):
                file_path = part[1].encode('utf-8')
            else:
                file_path = part[1]
            file_path = escape(file_path)
            xml_string += r'<Part>'
            xml_string += r'<PartNumber>' + str(part[0]) + r'</PartNumber>'
            xml_string += r'<PartName>' + str(file_path) + r'</PartName>'
            xml_string += r'<ETag>"' + str(part[2]).upper() + r'"</ETag>'
            xml_string += r'</Part>'
        else:
            print "the ", part, " in part_msg_list is not as expected!"
            return ""
    xml_string += r'</CreateFileGroup>'

    return xml_string

def create_object_link_msg_xml_by_name(object_list = None):
    '''
    get information from object_list and covert it to xml.
    '''
    if not object_list:
        object_list = []
    xml_string = r'<CreateObjectLink>'
    for i in range(len(object_list)):
        part = str(object_list[i]).strip()
        if isinstance(part, unicode):
            file_path = part.encode('utf-8')
        else:
            file_path = part
        file_path = escape(file_path)
        xml_string += r'<Part>'
        xml_string += r'<PartNumber>' + str(i + 1) + r'</PartNumber>'
        xml_string += r'<PartName>' + str(file_path) + r'</PartName>'
        xml_string += r'</Part>'
    xml_string += r'</CreateObjectLink>'

    return xml_string

def create_object_link_msg_xml(part_msg_list = None):
    '''
    get information from part_msg_list and covert it to xml.
    part_msg_list has special format.
    '''
    if not part_msg_list:
        part_msg_list = []
    xml_string = r'<CreateObjectLink>'
    for part in part_msg_list:
        if len(part) >= 2:
            if isinstance(part[1], unicode):
                file_path = part[1].encode('utf-8')
            else:
                file_path = part[1]
            file_path = escape(file_path)
            xml_string += r'<Part>'
            xml_string += r'<PartNumber>' + str(part[0]) + r'</PartNumber>'
            xml_string += r'<PartName>' + str(file_path) + r'</PartName>'
            xml_string += r'</Part>'
        else:
            print "the ", part, " in part_msg_list is not as expected!"
            return ""
    xml_string += r'</CreateObjectLink>'

    return xml_string

def create_part_xml(part_msg_list=None):
    '''
    get information from part_msg_list and covert it to xml.
    part_msg_list has special format.
    '''
    if not part_msg_list:
        part_msg_list = []
    xml_string = r'<CompleteMultipartUpload>'
    for part in part_msg_list:
        if len(part) >= 3:
            xml_string += r'<Part>'
            xml_string += r'<PartNumber>' + str(part[0]) + r'</PartNumber>'
            xml_string += r'<ETag>"' + str(part[2]).upper() + r'"</ETag>'
            xml_string += r'</Part>'
        else:
            print "the ", part, " in part_msg_list is not as expected!"
            return ""
    xml_string += r'</CompleteMultipartUpload>'

    return xml_string

def create_delete_object_msg_xml(object_list=None, is_quiet=False, is_defult=False):
    '''
    covert object name list to xml.
    '''
    if not object_list:
        object_list = []
    xml_string = r'<Delete>'
    if not is_defult:
        if is_quiet:
            xml_string += r'<Quiet>true</Quiet>'
        else:
            xml_string += r'<Quiet>false</Quiet>'
    for object in object_list:
        key = object.strip()
        if isinstance(object, unicode):
            key = object.encode('utf-8')
        key = escape(key)
        xml_string += r'<Object><Key>%s</Key></Object>' % key
    xml_string += r'</Delete>'
    return xml_string

############### operate OSS ###############
def clear_all_object_of_bucket(oss_instance, bucket):
    '''
    clean all objects in bucket, after that, it will delete this bucket.
    '''
    return clear_all_objects_in_bucket(oss_instance, bucket)

def clear_all_objects_in_bucket(oss_instance, bucket, delete_marker="", delete_upload_id_marker="", debug=False):
    '''
    it will clean all objects in bucket, after that, it will delete this bucket.

    example:
    from oss_api import *
    host = ""
    id = ""
    key = ""
    oss_instance = OssAPI(host, id, key)
    bucket = "leopublicreadprivatewrite"
    if clear_all_objects_in_bucket(oss_instance, bucket):
        pass
    else:
        print "clean Fail"
    '''
    prefix = ""
    delimiter = ""
    maxkeys = 1000
    delete_all_objects(oss_instance, bucket, prefix, delimiter, delete_marker, maxkeys, debug)
    delete_all_parts(oss_instance, bucket, delete_marker, delete_upload_id_marker, debug)
    res = oss_instance.delete_bucket(bucket)
    if (res.status / 100 != 2 and res.status != 404):
        print "clear_all_objects_in_bucket: delete bucket:%s fail, ret:%s, request id:%s" % (bucket, res.status, res.getheader("x-oss-request-id"))
        return False
    return True

def delete_all_objects(oss_instance, bucket, prefix="", delimiter="", delete_marker="", maxkeys=1000, debug=False):
    marker = delete_marker
    delete_obj_num = 0
    while 1:
        object_list = []
        res = oss_instance.get_bucket(bucket, prefix, marker, delimiter, maxkeys)
        if res.status != 200:
            break
        body = res.read()
        (tmp_object_list, marker) = get_object_list_marker_from_xml(body)
        for item in tmp_object_list:
            object_list.append(item[0])

        if object_list:
            object_list_xml = create_delete_object_msg_xml(object_list)
            res = oss_instance.batch_delete_object(bucket, object_list_xml)
            if res.status/100 != 2:
                if marker:
                    print "delete_all_objects: batch delete objects in bucket:%s fail, ret:%s, request id:%s, first object:%s, marker:%s" % (bucket, res.status, res.getheader("x-oss-request-id"), object_list[0], marker)
                else:
                    print "delete_all_objects: batch delete objects in bucket:%s fail, ret:%s, request id:%s, first object:%s" % (bucket, res.status, res.getheader("x-oss-request-id"), object_list[0])
            else:
                if debug:
                    delete_obj_num += len(object_list)
                    if marker:
                        print "delete_all_objects: Now %s objects deleted, marker:%s" % (delete_obj_num, marker)
                    else:
                        print "delete_all_objects: Now %s objects deleted" % (delete_obj_num)
        if len(marker) == 0:
            break

def delete_all_parts(oss_instance, bucket, delete_object_marker="", delete_upload_id_marker="", debug=False):
    delete_mulitipart_num = 0
    marker = delete_object_marker
    id_marker = delete_upload_id_marker
    while 1:
        res = oss_instance.get_all_multipart_uploads(bucket, key_marker=marker, upload_id_marker=id_marker)
        if res.status != 200:
            break
        body = res.read()
        hh = GetMultipartUploadsXml(body)
        (fl, pl) = hh.list()
        for i in fl:
            object = i[0]
            if isinstance(i[0], unicode):
                object = i[0].encode('utf-8')
            res = oss_instance.cancel_upload(bucket, object, i[1])
            if (res.status / 100 != 2 and res.status != 404):
                print "delete_all_parts: cancel upload object:%s, upload_id:%s FAIL, ret:%s, request-id:%s" % (object, i[1], res.status, res.getheader("x-oss-request-id"))
            else:
                delete_mulitipart_num += 1
                if debug:
                    print "delete_all_parts: cancel upload object:%s, upload_id:%s OK\nNow %s parts deleted." % (object, i[1], delete_mulitipart_num)
        if hh.is_truncated:
            marker = hh.next_key_marker
            id_marker = hh.next_upload_id_marker
        else:
            break
        if not marker:
            break

def clean_all_bucket(oss_instance):
    '''
    it will clean all bucket, including the all objects in bucket.
    '''
    res = oss_instance.get_service()
    if (res.status / 100) == 2:
        h = GetServiceXml(res.read())
        for b in h.bucket_list:
            if not clear_all_objects_in_bucket(oss_instance, b.name):
                print "clean bucket ", b.name, " failed! in clean_all_bucket"
                return False
        return True
    else:
        print "failed! get service in clean_all_bucket return ", res.status
        print res.read()
        print res.getheaders()
        return False

def pgfs_clear_all_objects_in_bucket(oss_instance, bucket):
    '''
    it will clean all objects in bucket, after that, it will delete this bucket.
    '''
    b = GetAllObjects()
    b.get_all_object_in_bucket(oss_instance, bucket)
    for i in b.object_list:
        res = oss_instance.delete_object(bucket, i)
        if (res.status / 100 != 2):
            print "clear_all_objects_in_bucket: delete object fail, ret is:", res.status, "bucket is:", bucket, "object is: ", i
            return False
        else:
            pass
    res = oss_instance.delete_bucket(bucket)
    if (res.status / 100 != 2 and res.status != 404):
        print "clear_all_objects_in_bucket: delete bucket fail, ret is: %s, request id is:%s" % (res.status, res.getheader("x-oss-request-id"))
        return False
    return True

def pgfs_clean_all_bucket(oss_instance):
    '''
    it will clean all bucket, including the all objects in bucket.
    '''
    res = oss_instance.get_service()
    if (res.status / 100) == 2:
        h = GetServiceXml(res.read())
        for b in h.bucket_list:
            if not pgfs_clear_all_objects_in_bucket(oss_instance, b.name):
                print "clean bucket ", b.name, " failed! in clean_all_bucket"
                return False
        return True
    else:
        print "failed! get service in clean_all_bucket return ", res.status
        print res.read()
        print res.getheaders()
        return False

def delete_all_parts_of_object_group(oss, bucket, object_group_name):
    res = oss.get_object_group_index(bucket, object_group_name)
    if res.status == 200:
        body = res.read()
        h = GetObjectGroupIndexXml(body)
        object_group_index = h.list()
        for i in object_group_index:
            if len(i) == 4 and len(i[1]) > 0:
                part_name = i[1].strip()
                res = oss.delete_object(bucket, part_name)
                if res.status != 204:
                    print "delete part ", part_name, " in bucket:", bucket, " failed!"
                    return False
    else:
        return False
    return True

def delete_all_parts_of_object_link(oss, bucket, object_link_name):
    res = oss.get_link_index(bucket, object_link_name)
    if res.status == 200:
        body = res.read()
        h = GetObjectLinkIndexXml(body)
        object_link_index = h.list()
        for i in object_link_index:
            if len(i) == 2 and len(i[1]) > 0:
                part_name = i[1].strip()
                res = oss.delete_object(bucket, part_name)
                if res.status != 204:
                    print "delete part ", part_name, " in bucket:", bucket, " failed!"
                    return False
    else:
        return False
    return True

class GetAllObjects:
    def __init__(self):
        self.object_list = []
        self.dir_list = []

    def get_object_in_bucket(self, oss, bucket="", marker="", prefix=""):
        object_list = []
        maxkeys = 1000
        try:
            res = oss.get_bucket(bucket, prefix, marker, maxkeys=maxkeys)
            body = res.read()
            hh = GetBucketXml(body)
            (fl, pl) = hh.list()
            if len(fl) != 0:
                for i in fl:
                    if isinstance(i[0], unicode):
                        object = i[0].encode('utf-8')
                    object_list.append(object)
            marker = hh.nextmarker
        except:
            pass
        return (object_list, marker)
    
    def get_object_dir_in_bucket(self, oss, bucket="", marker="", prefix="", delimiter=""):
        object_list = []
        dir_list = []
        maxkeys = 1000
        try:
            res = oss.get_bucket(bucket, prefix, marker, delimiter, maxkeys=maxkeys)
            body = res.read()
            hh = GetBucketXml(body)
            (fl, pl) = hh.list()
            if len(fl) != 0:
                for i in fl:
                    object_list.append((i[0], i[3], i[1]))  #name, size, modified_time
            if len(pl) != 0:
                for i in pl:
                    dir_list.append(i)
            marker = hh.nextmarker
        except:
            pass
        return (object_list, dir_list, marker)

    def get_all_object_in_bucket(self, oss, bucket="", marker="", prefix=""):
        marker2 = ""
        while True:
            (object_list, marker) = self.get_object_in_bucket(oss, bucket, marker2, prefix)
            marker2 = marker
            if len(object_list) != 0:
                self.object_list.extend(object_list)
            if not marker:
                break

    def get_all_object_dir_in_bucket(self, oss, bucket="", marker="", prefix="", delimiter=""):
        marker2 = ""
        while True:
            (object_list, dir_list, marker) = self.get_object_dir_in_bucket(oss, bucket, marker2, prefix, delimiter)
            marker2 = marker
            if len(object_list) != 0:
                self.object_list.extend(object_list)
            if len(dir_list) != 0:
                self.dir_list.extend(dir_list)
            if not marker:
                break
        return (self.object_list, self.dir_list)

def get_all_buckets(oss):
    bucket_list = []
    res = oss.get_service()
    if res.status == 200:
        h = GetServiceXml(res.read())
        for b in h.bucket_list:
            bucket_list.append(str(b.name).strip())
    return bucket_list 

def get_object_list_marker_from_xml(body):
    #return ([(object_name, object_length, last_modify_time)...], marker)
    object_meta_list = []
    next_marker = ""
    hh = GetBucketXml(body)
    (fl, pl) = hh.list()
    if len(fl) != 0:
        for i in fl:
            if isinstance(i[0], unicode):
                object = i[0].encode('utf-8')
            else:
                object = i[0]
            last_modify_time = i[1]
            length = i[3]
            etag = i[2]
            object_meta_list.append((object, length, last_modify_time, etag))
    if hh.is_truncated:
        next_marker = hh.nextmarker
    return (object_meta_list, next_marker)

def get_upload_id(oss, bucket, object, headers=None):
    '''
    get the upload id of object.
    Returns:
            string
    '''
    if not headers:
        headers = {}
    upload_id = ""
    res = oss.init_multi_upload(bucket, object, headers)
    if res.status == 200:
        body = res.read()
        h = GetInitUploadIdXml(body)
        upload_id = h.upload_id
    else:
        print res.status
        print res.getheaders()
        print res.read()
    return upload_id

def get_all_upload_id_list(oss, bucket):
    '''
    get all upload id of bucket
    Returns:
            list
    '''
    all_upload_id_list = []
    marker = ""
    id_marker = ""
    while True:
        res = oss.get_all_multipart_uploads(bucket, key_marker=marker, upload_id_marker=id_marker)
        if res.status != 200:
            return all_upload_id_list

        body = res.read()
        hh = GetMultipartUploadsXml(body)
        (fl, pl) = hh.list()
        for i in fl:
            all_upload_id_list.append(i)
        if hh.is_truncated:
            marker = hh.next_key_marker
            id_marker = hh.next_upload_id_marker
        else:
            break
        if not marker and not id_marker:
            break
    return all_upload_id_list

def get_upload_id_list(oss, bucket, object):
    '''
    get all upload id list of one object.
    Returns:
            list
    '''
    upload_id_list = []
    marker = ""
    id_marker = ""
    while True:
        res = oss.get_all_multipart_uploads(bucket, prefix=object, key_marker=marker, upload_id_marker=id_marker)
        if res.status != 200:
            break
        body = res.read()
        hh = GetMultipartUploadsXml(body)
        (fl, pl) = hh.list()
        for i in fl:
            upload_id_list.append(i[1])
        if hh.is_truncated:
            marker = hh.next_key_marker
            id_marker = hh.next_upload_id_marker
        else:
            break
        if not marker:
            break

    return upload_id_list

def get_part_list(oss, bucket, object, upload_id, max_part=""):
    '''
    get uploaded part list of object.
    Returns:
            list
    '''
    part_list = []
    marker = ""
    while True:
        res = oss.get_all_parts(bucket, object, upload_id, part_number_marker = marker, max_parts=max_part)
        if res.status != 200:
            break
        body = res.read()
        h = GetPartsXml(body)
        part_list.extend(h.list())
        if h.is_truncated:
            marker = h.next_part_number_marker
        else:
            break
        if not marker:
            break
    return part_list

def get_part_xml(oss, bucket, object, upload_id):
    '''
    get uploaded part list of object.
    Returns:
            string
    '''
    part_list = []
    part_list = get_part_list(oss, bucket, object, upload_id)
    xml_string = r'<CompleteMultipartUpload>'
    for part in part_list:
        xml_string += r'<Part>'
        xml_string += r'<PartNumber>' + str(part[0]) + r'</PartNumber>'
        xml_string += r'<ETag>' + part[1] + r'</ETag>'
        xml_string += r'</Part>'
    xml_string += r'</CompleteMultipartUpload>'
    return xml_string

def get_part_map(oss, bucket, object, upload_id):
    part_list = []
    part_list = get_part_list(oss, bucket, object, upload_id)
    part_map = {}
    for part in part_list:
        part_map[str(part[0])] = part[1]
    return part_map

########## multi-thread ##########
class DeleteObjectWorker(Thread):
    def __init__(self, oss, bucket, part_msg_list, retry_times=5):
        Thread.__init__(self)
        self.oss = oss
        self.bucket = bucket
        self.part_msg_list = part_msg_list
        self.retry_times = retry_times

    def run(self):
        bucket = self.bucket
        object_list = self.part_msg_list
        step = 1000
        begin = 0
        end = 0
        total_length = len(object_list)
        remain_length = total_length
        while True:
            if remain_length > step:
                end = begin + step
            elif remain_length > 0:
                end = begin + remain_length
            else:
                break
            is_fail = True
            retry_times = self.retry_times
            while True:
                try:
                    if retry_times <= 0:
                        break
                    res = self.oss.delete_objects(bucket, object_list[begin:end])
                    if res.status / 100 == 2:
                        is_fail = False
                        break
                except:
                    retry_times = retry_times - 1
                    time.sleep(1)
            if is_fail:
                print "delete object_list[%s:%s] failed!, first is %s" % (begin, end, object_list[begin])
            begin = end
            remain_length = remain_length - step

class PutObjectGroupWorker(Thread):
    def __init__(self, oss, bucket, file_path, part_msg_list, retry_times=5):
        Thread.__init__(self)
        self.oss = oss
        self.bucket = bucket
        self.part_msg_list = part_msg_list
        self.file_path = file_path
        self.retry_times = retry_times

    def run(self):
        for part in self.part_msg_list:
            if len(part) == 5:
                bucket = self.bucket
                file_name = part[1]
                if isinstance(file_name, unicode):
                    filename = file_name.encode('utf-8')
                object_name = file_name
                retry_times = self.retry_times
                is_skip = False
                while True:
                    try:
                        if retry_times <= 0:
                            break
                        res = self.oss.head_object(bucket, object_name)
                        if res.status == 200:
                            header_map = convert_header2map(res.getheaders())
                            etag = safe_get_element("etag", header_map)
                            md5 = part[2]
                            if etag.replace('"', "").upper() == md5.upper():
                                is_skip = True
                        break
                    except:
                        retry_times = retry_times - 1
                        time.sleep(1)

                if is_skip:
                    continue

                partsize = part[3]
                offset = part[4]
                retry_times = self.retry_times
                while True:
                    try:
                        if retry_times <= 0:
                            break
                        res = self.oss.put_object_from_file_given_pos(bucket, object_name, self.file_path, offset, partsize)
                        if res.status != 200:
                            print "upload ", file_name, "failed!", " ret is:", res.status
                            print "headers", res.getheaders()
                            retry_times = retry_times - 1
                            time.sleep(1)
                        else:
                            break
                    except:
                        retry_times = retry_times - 1
                        time.sleep(1)

            else:
                print "ERROR! part", part , " is not as expected!"

class PutObjectLinkWorker(Thread):
    def __init__(self, oss, bucket, file_path, part_msg_list, retry_times=5):
        Thread.__init__(self)
        self.oss = oss
        self.bucket = bucket
        self.part_msg_list = part_msg_list
        self.file_path = file_path
        self.retry_times = retry_times

    def run(self):
        for part in self.part_msg_list:
            if len(part) == 5:
                bucket = self.bucket
                file_name = part[1]
                if isinstance(file_name, unicode):
                    filename = file_name.encode('utf-8')
                object_name = file_name
                retry_times = self.retry_times
                is_skip = False
                while True:
                    try:
                        if retry_times <= 0:
                            break
                        res = self.oss.head_object(bucket, object_name)
                        if res.status == 200:
                            header_map = convert_header2map(res.getheaders())
                            etag = safe_get_element("etag", header_map)
                            md5 = part[2]
                            if etag.replace('"', "").upper() == md5.upper():
                                is_skip = True
                        break
                    except:
                        retry_times = retry_times - 1
                        time.sleep(1)

                if is_skip:
                    continue

                partsize = part[3]
                offset = part[4]
                retry_times = self.retry_times
                while True:
                    try:
                        if retry_times <= 0:
                            break
                        res = self.oss.put_object_from_file_given_pos(bucket, object_name, self.file_path, offset, partsize)
                        if res.status != 200:
                            print "upload ", file_name, "failed!", " ret is:", res.status
                            print "headers", res.getheaders()
                            retry_times = retry_times - 1
                            time.sleep(1)
                        else:
                            break
                    except:
                        retry_times = retry_times - 1
                        time.sleep(1)

            else:
                print "ERROR! part", part , " is not as expected!"

class UploadPartWorker(Thread):
    def __init__(self, oss, bucket, object, upoload_id, file_path, part_msg_list, uploaded_part_map, retry_times=5, debug=DEBUG):
        Thread.__init__(self)
        self.oss = oss
        self.bucket = bucket
        self.object = object
        self.part_msg_list = part_msg_list
        self.file_path = file_path
        self.upload_id = upoload_id
        self.uploaded_part_map = uploaded_part_map
        self.retry_times = retry_times

    def run(self):
        for part in self.part_msg_list:
            part_number = str(part[0])
            if len(part) == 5:
                bucket = self.bucket
                object = self.object
                if self.uploaded_part_map.has_key(part_number):
                    md5 = part[2]
                    if self.uploaded_part_map[part_number].replace('"', "").upper() == md5.upper():
                        continue

                partsize = part[3]
                offset = part[4]
                retry_times = self.retry_times
                while True:
                    try:
                        if retry_times <= 0:
                            break
                        res = self.oss.upload_part_from_file_given_pos(bucket, object, self.file_path, offset, partsize, self.upload_id, part_number)
                        if res.status != 200:
                            retry_times = retry_times - 1
                            time.sleep(1)
                        else:
                            break
                    except:
                        retry_times = retry_times - 1
                        time.sleep(1)
            else:
                pass

class MultiGetWorker(Thread):
    def __init__(self, oss, bucket, object, file, start, end, retry_times=5):
        Thread.__init__(self)
        self.oss = oss
        self.bucket = bucket
        self.object = object
        self.startpos = start
        self.endpos = end
        self.file = file
        self.length = self.endpos - self.startpos + 1
        self.need_read = 0
        self.get_buffer_size = 10*1024*1024
        self.retry_times = retry_times

    def run(self):
        if self.startpos >= self.endpos:
            return

        retry_times = 0
        while True:
            headers = {}
            self.file.seek(self.startpos)
            headers['Range'] = 'bytes=%d-%d' % (self.startpos, self.endpos)
            try:
                res = self.oss.object_operation("GET", self.bucket, self.object, headers)
                if res.status == 206:
                    while self.need_read < self.length:
                        left_len = self.length - self.need_read
                        if left_len > self.get_buffer_size:
                            content = res.read(self.get_buffer_size)
                        else:
                            content = res.read(left_len)
                        if content:
                            self.need_read += len(content)
                            self.file.write(content)
                        else:
                            break
                    break
            except:
                pass
            retry_times += 1
            if retry_times > self.retry_times:
                print "ERROR, reach max retry times:%s when multi get /%s/%s" % (self.retry_times, self.bucket, self.object)
                break 

        self.file.flush()
        self.file.close()

############### misc ###############

def split_large_file(file_path, object_prefix="", max_part_num=1000, part_size=10*1024*1024, buffer_size=10*1024):
    parts_list = []

    if os.path.isfile(file_path):
        file_size = os.path.getsize(file_path)

        if file_size > part_size * max_part_num:
            part_size = (file_size + max_part_num - file_size % max_part_num) / max_part_num

        part_order = 1
        fp = open(file_path, 'rb')
        fp.seek(os.SEEK_SET)

        part_num = (file_size + part_size - 1) / part_size

        for i in xrange(0, part_num):
            left_len = part_size
            real_part_size = 0
            m = md5.new()
            offset = part_size * i
            while True:
                read_size = 0
                if left_len <= 0:
                    break
                elif left_len < buffer_size:
                    read_size = left_len
                else:
                    read_size = buffer_size

                buffer_content = fp.read(read_size)
                m.update(buffer_content)
                real_part_size += len(buffer_content)

                left_len = left_len - read_size

            md5sum = m.hexdigest()

            temp_file_name = os.path.basename(file_path) + "_" + str(part_order)
            if isinstance(object_prefix, unicode):
                object_prefix = object_prefix.encode('utf-8')
            if not object_prefix:
                file_name = sum_string(temp_file_name) + "_" + temp_file_name
            else:
                file_name = object_prefix + "/" + sum_string(temp_file_name) + "_" + temp_file_name
            part_msg = (part_order, file_name, md5sum, real_part_size, offset)
            parts_list.append(part_msg)
            part_order += 1

        fp.close()
    else:
        print "ERROR! No file: ", file_path, ", please check."

    return parts_list

def sumfile(fobj):
    '''Returns an md5 hash for an object with read() method.'''
    m = md5.new()
    while True:
        d = fobj.read(8096)
        if not d:
            break
        m.update(d)
    return m.hexdigest()

def md5sum(fname):
    '''Returns an md5 hash for file fname, or stdin if fname is "-".'''
    if fname == '-':
        ret = sumfile(sys.stdin)
    else:
        try:
            f = file(fname, 'rb')
        except:
            return 'Failed to open file'
        ret = sumfile(f)
        f.close()
    return ret

def md5sum2(filename, offset=0, partsize=0):
    m = md5.new()
    fp = open(filename, 'rb')
    if offset > os.path.getsize(filename):
        fp.seek(os.SEEK_SET, os.SEEK_END)
    else:
        fp.seek(offset)

    left_len = partsize
    BufferSize = 8 * 1024
    while True:
        if left_len <= 0:
            break
        elif left_len < BufferSize:
            buffer_content = fp.read(left_len)
        else:
            buffer_content = fp.read(BufferSize)
        m.update(buffer_content)
        left_len = left_len - len(buffer_content)
    md5sum = m.hexdigest()
    return md5sum

def sum_string(content):
    f = StringIO.StringIO(content)
    md5sum = sumfile(f)
    f.close()
    return md5sum

def convert_header2map(header_list):
    header_map = {}
    for (a, b) in header_list:
        header_map[a] = b
    return header_map

def safe_get_element(name, container):
    for k, v in container.items():
        if k.strip().lower() == name.strip().lower():
            return v
    return ""

def get_content_type_by_filename(file_name):
    mime_type = ""
    try:
        suffix = ""
        name = os.path.basename(file_name)
        suffix = name.split('.')[-1]
        if suffix == "js":
            mime_type = "application/javascript"
        import mimetypes
        mimetypes.init()
        mime_type = mimetypes.types_map["." + suffix]
    except Exception:
        mime_type = 'application/octet-stream'
    return mime_type

def smart_code(input_stream):
    if isinstance(input_stream, str):
        try:
            tmp = unicode(input_stream, 'utf-8')
        except UnicodeDecodeError:
            try:
                tmp = unicode(input_stream, 'gbk')
            except UnicodeDecodeError:
                try:
                    tmp = unicode(input_stream, 'big5')
                except UnicodeDecodeError:
                    try:
                        tmp = unicode(input_stream, 'ascii')
                    except:
                        tmp = input_stream
    else:
        tmp = input_stream
    return tmp

def is_ip(s):
    try:
        tmp_list = s.split(':')
        s = tmp_list[0]
        if s == 'localhost':
            return True
        tmp_list = s.split('.')
        if len(tmp_list) != 4:
            return False
        else:
            for i in tmp_list:
                if int(i) < 0 or int(i) > 255:
                    return False
    except:
        return False
    return True

def get_host_from_list(hosts):
    tmp_list = hosts.split(",")
    if len(tmp_list) <= 1:
        return get_second_level_domain(hosts)
    for tmp_host in tmp_list:
        tmp_host = tmp_host.strip()
        host = tmp_host
        port = 80
        try:
            host_port_list = tmp_host.split(":")
            if len(host_port_list) == 1:
                host = host_port_list[0].strip()
            elif len(host_port_list) == 2:
                host = host_port_list[0].strip()
                port = int(host_port_list[1].strip())
            sock=socket.socket(socket.AF_INET,socket.SOCK_STREAM)
            sock.connect((host, port))
            return get_second_level_domain(host)
        except:
            pass
    return get_second_level_domain(tmp_list[0].strip())
    
def get_second_level_domain(host):
    if is_ip(host):
        return host
    else:
        tmp_list = host.split('.')
        if len(tmp_list) >= 4:
            return ".".join(tmp_list[-3:])
    return host

if __name__ == '__main__':
    pass
