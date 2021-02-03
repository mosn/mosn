/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package hessian

import (
	"github.com/apache/dubbo-go-hessian2/java_exception"
)

func init() {
	RegisterPOJO(&java_exception.Method{})
	RegisterPOJO(&java_exception.Class{})
	RegisterPOJO(&java_exception.Throwable{})
	RegisterPOJO(&java_exception.Exception{})
	RegisterPOJO(&java_exception.IOException{})
	RegisterPOJO(&java_exception.RuntimeException{})
	RegisterPOJO(&java_exception.StackTraceElement{})
	RegisterPOJO(&java_exception.ClassCastException{})
	RegisterPOJO(&java_exception.ArrayStoreException{})
	RegisterPOJO(&java_exception.IllegalStateException{})
	RegisterPOJO(&java_exception.IllegalMonitorStateException{})
	RegisterPOJO(&java_exception.EnumConstantNotPresentException{})
	RegisterPOJO(&java_exception.CloneNotSupportedException{})
	RegisterPOJO(&java_exception.InterruptedException{})
	RegisterPOJO(&java_exception.InterruptedIOException{})
	RegisterPOJO(&java_exception.LambdaConversionException{})
	RegisterPOJO(&java_exception.UnmodifiableClassException{})
	RegisterPOJO(&java_exception.MalformedParameterizedTypeException{})
	RegisterPOJO(&java_exception.MalformedParametersException{})
	RegisterPOJO(&java_exception.TypeNotPresentException{})
	RegisterPOJO(&java_exception.UndeclaredThrowableException{})
	RegisterPOJO(&java_exception.WrongMethodTypeException{})
	RegisterPOJO(&java_exception.NullPointerException{})
	RegisterPOJO(&java_exception.UncheckedIOException{})
	RegisterPOJO(&java_exception.FileNotFoundException{})
	RegisterPOJO(&java_exception.EOFException{})
	RegisterPOJO(&java_exception.SyncFailedException{})
	RegisterPOJO(&java_exception.ObjectStreamException{})
	RegisterPOJO(&java_exception.WriteAbortedException{})
	RegisterPOJO(&java_exception.InvalidObjectException{})
	RegisterPOJO(&java_exception.StreamCorruptedException{})
	RegisterPOJO(&java_exception.InvalidClassException{})
	RegisterPOJO(&java_exception.OptionalDataException{})
	RegisterPOJO(&java_exception.NotActiveException{})
	RegisterPOJO(&java_exception.NotSerializableException{})
	RegisterPOJO(&java_exception.UTFDataFormatException{})
	RegisterPOJO(&java_exception.SecurityException{})
	RegisterPOJO(&java_exception.IllegalArgumentException{})
	RegisterPOJO(&java_exception.IllegalThreadStateException{})
	RegisterPOJO(&java_exception.NumberFormatException{})
	RegisterPOJO(&java_exception.IndexOutOfBoundsException{})
	RegisterPOJO(&java_exception.ArrayIndexOutOfBoundsException{})
	RegisterPOJO(&java_exception.StringIndexOutOfBoundsException{})
	RegisterPOJO(&java_exception.IllegalFormatWidthException{})
	RegisterPOJO(&java_exception.IllegalFormatConversionException{})
	RegisterPOJO(&java_exception.DuplicateFormatFlagsException{})
	RegisterPOJO(&java_exception.MissingResourceException{})
	RegisterPOJO(&java_exception.ConcurrentModificationException{})
	RegisterPOJO(&java_exception.RejectedExecutionException{})
	RegisterPOJO(&java_exception.CompletionException{})
	RegisterPOJO(&java_exception.EmptyStackException{})
	RegisterPOJO(&java_exception.IllformedLocaleException{})
	RegisterPOJO(&java_exception.NoSuchElementException{})
	RegisterPOJO(&java_exception.NegativeArraySizeException{})
	RegisterPOJO(&java_exception.UnsupportedOperationException{})
	RegisterPOJO(&java_exception.ArithmeticException{})
	RegisterPOJO(&java_exception.InputMismatchException{})
	RegisterPOJO(&java_exception.ExecutionException{})
	RegisterPOJO(&java_exception.InvalidPreferencesFormatException{})
	RegisterPOJO(&java_exception.TimeoutException{})
	RegisterPOJO(&java_exception.BackingStoreException{})
	RegisterPOJO(&java_exception.DataFormatException{})
	RegisterPOJO(&java_exception.BrokenBarrierException{})
	RegisterPOJO(&java_exception.TooManyListenersException{})
	RegisterPOJO(&java_exception.InvalidPropertiesFormatException{})
	RegisterPOJO(&java_exception.ZipException{})
	RegisterPOJO(&java_exception.JarException{})
	RegisterPOJO(&java_exception.IllegalClassFormatException{})
	RegisterPOJO(&java_exception.ReflectiveOperationException{})
	RegisterPOJO(&java_exception.InvocationTargetException{})
	RegisterPOJO(&java_exception.NoSuchMethodException{})
	RegisterPOJO(&java_exception.NoSuchFieldException{})
	RegisterPOJO(&java_exception.IllegalAccessException{})
	RegisterPOJO(&java_exception.ClassNotFoundException{})
	RegisterPOJO(&java_exception.InstantiationException{})
	RegisterPOJO(&java_exception.DateTimeException{})
	RegisterPOJO(&java_exception.UnsupportedTemporalTypeException{})
	RegisterPOJO(&java_exception.ZoneRulesException{})
	RegisterPOJO(&java_exception.DateTimeParseException{})
	RegisterPOJO(&java_exception.FormatterClosedException{})
	RegisterPOJO(&java_exception.CancellationException{})
	RegisterPOJO(&java_exception.UnknownFormatConversionException{})
	RegisterPOJO(&java_exception.UnknownFormatFlagsException{})
	RegisterPOJO(&java_exception.IllegalFormatFlagsException{})
	RegisterPOJO(&java_exception.IllegalFormatPrecisionException{})
	RegisterPOJO(&java_exception.IllegalFormatCodePointException{})
	RegisterPOJO(&java_exception.MissingFormatArgumentException{})
	RegisterPOJO(&java_exception.MissingFormatWidthException{})
	RegisterPOJO(&java_exception.DubboGenericException{})
	RegisterPOJO(&java_exception.IncompleteAnnotationException{})
	RegisterPOJO(&java_exception.AnnotationTypeMismatchException{})
}
