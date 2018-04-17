package javaio

const (
	/**
   * Magic number that is written to the stream header.
   */
	// final static short STREAM_MAGIC = (short)0xaced;
	STREAM_MAGIC = 0xaced

	/**
	 * Version number that is written to the stream header.
	 */
	// final static short STREAM_VERSION = 5; 0x0005
	STREAM_VERSION = 5

	/* Each item in the stream is preceded by a tag
	 */

	/**
	 * First tag value.
	 */
	// final static byte TC_BASE = 0x70;
	TC_BASE = 0x70

	/**
	 * Null object reference.
	 */
	// final static byte TC_NULL =         (byte)0x70;
	TC_NULL = 0x70

	/**
	 * Reference to an object already written into the stream.
	 */
	// final static byte TC_REFERENCE =    (byte)0x71;
	TC_REFERENCE = 0x71

	/**
	 * new Class Descriptor.
	 */
	// final static byte TC_CLASSDESC =    (byte)0x72;
	TC_CLASSDESC = 0x72

	/**
	 * new Object.
	 */
	// final static byte TC_OBJECT =       (byte)0x73;
	TC_OBJECT = 0x73

	/**
	 * new String.
	 */
	// final static byte TC_STRING =       (byte)0x74; 116
	TC_STRING = 0x74

	/**
	 * new Array.
	 */
	// final static byte TC_ARRAY =        (byte)0x75;
	TC_ARRAY = 0x75

	/**
	 * Reference to Class.
	 */
	// final static byte TC_CLASS =        (byte)0x76;
	TC_CLASS = 0x76

	/**
	 * Block of optional data. Byte following tag indicates number
	 * of bytes in this block data.
	 */
	// final static byte TC_BLOCKDATA =    (byte)0x77;
	TC_BLOCKDATA = 0x77

	/**
	 * End of optional block data blocks for an object.
	 */
	// final static byte TC_ENDBLOCKDATA = (byte)0x78;
	TC_ENDBLOCKDATA = 0x78

	/**
	 * Reset stream context. All handles written into stream are reset.
	 */
	// final static byte TC_RESET =        (byte)0x79;
	TC_RESET = 0x79

	/**
	 * long Block data. The long following the tag indicates the
	 * number of bytes in this block data.
	 */
	// final static byte TC_BLOCKDATALONG= (byte)0x7A;
	TC_BLOCKDATALONG = 0x7A

	/**
	 * Exception during write.
	 */
	// final static byte TC_EXCEPTION =    (byte)0x7B;
	TC_EXCEPTION = 0x7B

	/**
	 * Long string.
	 */
	// final static byte TC_LONGSTRING =   (byte)0x7C;
	TC_LONGSTRING = 0x7C

	/**
	 * new Proxy Class Descriptor.
	 */
	// final static byte TC_PROXYCLASSDESC =       (byte)0x7D;
	TC_PROXYCLASSDESC = 0x7D

	/**
	 * new Enum constant.
	 * @since 1.5
	 */
	// final static byte TC_ENUM =         (byte)0x7E;
	TC_ENUM = 0x7E

	/**
	 * Last tag value.
	 */
	// final static byte TC_MAX =          (byte)0x7E;
	TC_MAX = 0x7E

	/**
	 * First wire handle to be assigned.
	 */
	// final static int baseWireHandle = 0x7e0000;
	baseWireHandle = 0x7e0000

	/******************************************************/
	/* Bit masks for ObjectStreamClass flag.*/

	/**
	 * Bit mask for ObjectStreamClass flag. Indicates a Serializable class
	 * defines its own writeObject method.
	 */
	// final static byte SC_WRITE_METHOD = 0x01;
	SC_WRITE_METHOD = 0x01

	/**
	 * Bit mask for ObjectStreamClass flag. Indicates Externalizable data
	 * written in Block Data mode.
	 * Added for PROTOCOL_VERSION_2.
	 *
	 * @see #PROTOCOL_VERSION_2
	 * @since 1.2
	 */
	// final static byte SC_BLOCK_DATA = 0x08;
	SC_BLOCK_DATA = 0x08

	/**
	 * Bit mask for ObjectStreamClass flag. Indicates class is Serializable.
	 */
	// final static byte SC_SERIALIZABLE = 0x02; // meaing java.io.Serializable
	SC_SERIALIZABLE = 0x02

	/**
	 * Bit mask for ObjectStreamClass flag. Indicates class is Externalizable.
	 */
	// final static byte SC_EXTERNALIZABLE = 0x04;
	SC_EXTERNALIZABLE = 0x04

	/**
	 * Bit mask for ObjectStreamClass flag. Indicates class is an enum type.
	 * @since 1.5
	 */
	// final static byte SC_ENUM = 0x10;
	SC_ENUM = 0x10

	/* *******************************************************************/
	/* Security permissions */

	/**
	 * Enable substitution of one object for another during
	 * serialization/deserialization.
	 *
	 * @see java.io.ObjectOutputStream#enableReplaceObject(boolean)
	 * @see java.io.ObjectInputStream#enableResolveObject(boolean)
	 * @since 1.2
	 */
	// final static SerializablePermission SUBSTITUTION_PERMISSION =
	//                        new SerializablePermission("enableSubstitution");

	/**
	 * Enable overriding of readObject and writeObject.
	 *
	 * @see java.io.ObjectOutputStream#writeObjectOverride(Object)
	 * @see java.io.ObjectInputStream#readObjectOverride()
	 * @since 1.2
	 */
	// final static SerializablePermission SUBCLASS_IMPLEMENTATION_PERMISSION =
	//                 new SerializablePermission("enableSubclassImplementation");
	/**
	 * A Stream Protocol Version. <p>
	 *
	 * All externalizable data is written in JDK 1.1 external data
	 * format after calling this method. This version is needed to write
	 * streams containing Externalizable data that can be read by
	 * pre-JDK 1.1.6 JVMs.
	 *
	 * @see java.io.ObjectOutputStream#useProtocolVersion(int)
	 * @since 1.2
	 */
	// public final static int PROTOCOL_VERSION_1 = 1;
	PROTOCOL_VERSION_1 = 1

	/**
	 * A Stream Protocol Version. <p>
	 *
	 * This protocol is written by JVM 1.2.
	 *
	 * Externalizable data is written in block data mode and is
	 * terminated with TC_ENDBLOCKDATA. Externalizable classdescriptor
	 * flags has SC_BLOCK_DATA enabled. JVM 1.1.6 and greater can
	 * read this format change.
	 *
	 * Enables writing a nonSerializable class descriptor into the
	 * stream. The serialVersionUID of a nonSerializable class is
	 * set to 0L.
	 *
	 * @see java.io.ObjectOutputStream#useProtocolVersion(int)
	 * @see #SC_BLOCK_DATA
	 * @since 1.2
	 */
	// public final static int PROTOCOL_VERSION_2 = 2;
	PROTOCOL_VERSION_2 = 2
)
